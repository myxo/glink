#include <boost/asio.hpp>
#include <cstdlib>
#include <deque>
#include <list>
#include <memory>
#include <set>
#include <utility>

#include "message.h"
#include "network.h"

#include <spdlog/spdlog.h>

using boost::asio::ip::tcp;

//----------------------------------------------------------------------

typedef std::deque<Message> chat_message_queue;

//----------------------------------------------------------------------

class chat_participant {
public:
    virtual ~chat_participant() {}
    virtual void deliver(const Message& msg) = 0;
};

typedef std::shared_ptr<chat_participant> chat_participant_ptr;

//----------------------------------------------------------------------

class chat_room {
public:
    void join(chat_participant_ptr participant) {
        participants_.insert(participant);
        for (auto msg : recent_msgs_) participant->deliver(msg);
    }

    void leave(chat_participant_ptr participant) { participants_.erase(participant); }

    void deliver(const Message& msg) {
        recent_msgs_.push_back(msg);
        while (recent_msgs_.size() > max_recent_msgs) recent_msgs_.pop_front();

        for (auto participant : participants_) participant->deliver(msg);
    }

private:
    std::set<chat_participant_ptr> participants_;
    enum { max_recent_msgs = 100 };
    chat_message_queue recent_msgs_;
};

//----------------------------------------------------------------------

class chat_session : public chat_participant, public std::enable_shared_from_this<chat_session> {
public:
    chat_session(tcp::socket socket, chat_room& room) : socket_(std::move(socket)), room_(room) {}

    void start() {
        room_.join(shared_from_this());
        do_read_header();
    }

    void deliver(const Message& msg) {
        bool write_in_progress = !write_msgs_.empty();
        write_msgs_.push_back(msg);
        if (!write_in_progress) {
            do_write();
        }
    }

private:
    void do_read_header() {
        auto self(shared_from_this());
        boost::asio::async_read(socket_, boost::asio::buffer(read_msg_.data(), Message::kHeaderSize),
                                [this, self](boost::system::error_code ec, std::size_t length) {
                                    if (!ec && read_msg_.decode_header()) {
                                        spdlog::trace("Server: read header ({} bytes), now async read body ({} bytes)",
                                                      length, read_msg_.body_length());
                                        do_read_body();
                                    } else {
                                        spdlog::info("Server: error on reading header: {}", ec.message());
                                        room_.leave(shared_from_this());
                                    }
                                });
    }

    void do_read_body() {
        auto self(shared_from_this());
        boost::asio::async_read(socket_, boost::asio::buffer(read_msg_.body(), read_msg_.body_length()),
                                [this, self](boost::system::error_code ec, std::size_t length) {
                                    if (!ec) {
                                        spdlog::debug("Server: finish read body ({} bytes): {}", length, read_msg_.body());
                                        room_.deliver(read_msg_);
                                        do_read_header();
                                    } else {
                                        spdlog::info("Server: error on reading body: {}", ec.message());
                                        room_.leave(shared_from_this());
                                    }
                                });
    }

    void do_write() {
        auto self(shared_from_this());
        boost::asio::async_write(socket_, boost::asio::buffer(write_msgs_.front().data(), write_msgs_.front().get_length()),
                                 [this, self](boost::system::error_code ec, std::size_t /*length*/) {
                                     if (!ec) {
                                         write_msgs_.pop_front();
                                         if (!write_msgs_.empty()) {
                                             do_write();
                                         }
                                     } else {
                                         room_.leave(shared_from_this());
                                     }
                                 });
    }

    tcp::socket socket_;
    chat_room& room_;
    Message read_msg_;
    chat_message_queue write_msgs_;
};

//----------------------------------------------------------------------

class chat_server {
public:
    chat_server(boost::asio::io_context& io_context, const tcp::endpoint& endpoint) : acceptor_(io_context, endpoint) {
        do_accept();
    }

public:
    void do_accept() {
        acceptor_.async_accept([this](boost::system::error_code ec, tcp::socket socket) {
            spdlog::debug("Server: accept new connection");
            if (!ec) {
                std::make_shared<chat_session>(std::move(socket), room_)->start();
            }

            do_accept();
        });
    }

    tcp::acceptor acceptor_;
    chat_room room_;
};

Server::Server() : endpoint_(tcp::v4(), 0) {
    impl_ = std::make_unique<chat_server>(io_context_, endpoint_);
}

Server::~Server() = default;

void Server::Start() { io_context_.run(); }
void Server::Stop() {}

std::string Server::GetIp() const {
    return impl_->acceptor_.local_endpoint().address().to_string();
}

uint16_t Server::GetPort() const {
    return impl_->acceptor_.local_endpoint().port();
}
