#include <spdlog/spdlog.h>

#include <boost/asio.hpp>
#include <cstdlib>
#include <deque>
#include <list>
#include <memory>
#include <set>
#include <utility>

#include "message.h"
#include "network.h"

using boost::asio::ip::tcp;

namespace net = boost::asio;

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
        for (auto msg : recent_msgs_) {
            participant->deliver(msg);
        }
    }

    void leave(chat_participant_ptr participant) {
        participants_.erase(participant);
    }

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
    chat_session(tcp::socket socket, chat_room& room)
        : socket_(std::move(socket)), room_(room), timer_(socket_.get_executor()) {
        timer_.expires_at(std::chrono::steady_clock::time_point::max());
    }

    void start() {
        room_.join(shared_from_this());

        co_spawn(
            socket_.get_executor(), [self = shared_from_this()] { return self->Reader(); }, net::detached);

        co_spawn(
            socket_.get_executor(), [self = shared_from_this()] { return self->Writer(); }, net::detached);
    }

    void deliver(const Message& msg) {
        write_msgs_.push_back(msg);
        timer_.cancel_one();
    }

    void Stop() {
        room_.leave(shared_from_this());
        socket_.close();
        timer_.cancel();
    }

private:
    net::awaitable<void> Reader() try {
        while (true) {
            Message read_msg;
            std::size_t length = co_await net::async_read(
                socket_, boost::asio::buffer(read_msg.data(), Message::kHeaderSize), net::use_awaitable);

            if (!read_msg.decode_header()) {
                spdlog::info("Server: error on reading header, cannot decode");
                room_.leave(shared_from_this());
                co_return;
            }
            spdlog::trace("Server: read header ({} bytes), now async read body ({} bytes)", length,
                          read_msg.body_length());

            length = co_await net::async_read(socket_, net::buffer(read_msg.body(), read_msg.body_length()),
                                              net::use_awaitable);

            spdlog::debug("Server: finish read body ({} bytes): {}", length, read_msg.body());
            spdlog::get("chat_msg")->info("{}", read_msg.body());

            room_.deliver(read_msg);
        }
    } catch (std::exception& e) {
        spdlog::warn("Exception in Reader: {}", e.what());
    }

    net::awaitable<void> Writer() try {
        while (socket_.is_open()) {
            if (write_msgs_.empty()) {
                boost::system::error_code ec;
                co_await timer_.async_wait(net::redirect_error(net::use_awaitable, ec));
            } else {
                co_await net::async_write(socket_,
                                          net::buffer(write_msgs_.front().data(), write_msgs_.front().get_length()),
                                          net::use_awaitable);
                write_msgs_.pop_front();
            }
        }

    } catch (std::exception& e) {
        spdlog::warn("Exception in Writer: {}", e.what());
    }

    tcp::socket socket_;
    chat_room& room_;
    chat_message_queue write_msgs_;
    net::steady_timer timer_;
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

void Server::Start() {
    io_context_.run();
}
void Server::Stop() {}

std::string Server::GetIp() const {
    return impl_->acceptor_.local_endpoint().address().to_string();
}

uint16_t Server::GetPort() const {
    return impl_->acceptor_.local_endpoint().port();
}
