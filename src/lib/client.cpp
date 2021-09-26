#include <boost/asio.hpp>
#include <cstdlib>
#include <deque>
#include <iostream>
#include <thread>

#include "message.h"
#include "network.h"

#include <spdlog/spdlog.h>

using boost::asio::ip::tcp;

typedef std::deque<Message> chat_message_queue;

class chat_client {
public:
    chat_client(boost::asio::io_context& io_context, const tcp::resolver::results_type& endpoints)
        : io_context_(io_context), socket_(io_context) {
        do_connect(endpoints);
    }

    void write(const Message& msg) {
        boost::asio::post(io_context_, [this, msg]() {
            bool write_in_progress = !write_msgs_.empty();
            write_msgs_.push_back(msg);
            if (!write_in_progress) {
                do_write();
            }
        });
    }

    void close() {
        boost::asio::post(io_context_, [this]() { socket_.close(); });
    }

private:
    void do_connect(const tcp::resolver::results_type& endpoints) {
        boost::asio::async_connect(socket_, endpoints, [this](boost::system::error_code ec, tcp::endpoint) {
            if (!ec) {
                do_read_header();
            }
        });
    }

    void do_read_header() {
        boost::asio::async_read(socket_, boost::asio::buffer(read_msg_.data(), Message::kHeaderSize),
                                [this](boost::system::error_code ec, std::size_t /*length*/) {
                                    if (!ec && read_msg_.decode_header()) {
                                        do_read_body();
                                    } else {
                                        socket_.close();
                                    }
                                });
    }

    void do_read_body() {
        boost::asio::async_read(socket_, boost::asio::buffer(read_msg_.body(), read_msg_.body_length()),
                                [this](boost::system::error_code ec, std::size_t /*length*/) {
                                    if (!ec) {
                                        //std::cout.write(read_msg_.body(), read_msg_.body_length());
                                        //std::cout << "\n";
                                        do_read_header();
                                    } else {
                                        socket_.close();
                                    }
                                });
    }

    void do_write() {
        boost::asio::async_write(socket_,
                                 boost::asio::buffer(write_msgs_.front().data(), write_msgs_.front().get_length()),
                                 [this](boost::system::error_code ec, std::size_t length) {
                                    spdlog::trace("Connection: write complete, wrote {} bytes", length);
                                     if (!ec) {
                                         write_msgs_.pop_front();
                                         if (!write_msgs_.empty()) {
                                             do_write();
                                         }
                                     } else {
                                         socket_.close();
                                     }
                                 });
    }

private:
    boost::asio::io_context& io_context_;
    tcp::socket socket_;
    Message read_msg_;
    chat_message_queue write_msgs_;
};

class Connection : public IConnection {
public:
    Connection(std::string ip, uint16_t port) {
        tcp::resolver resolver(io_context_);
        auto endpoints = resolver.resolve(ip.c_str(), std::to_string(port));
        client_.emplace(io_context_, endpoints);
        thread_ = std::jthread([this]() { 
            io_context_.run(); 
        });
    }

    ~Connection() {
        client_->close();
        thread_.join();
    }

    void SendMesasge(std::string msg) override {
        client_->write(Message{msg});
    }

private:
    std::jthread thread_;
    boost::asio::io_context io_context_;
    std::optional<chat_client> client_;
};

std::shared_ptr<IConnection> CreateConnection(std::string ip, uint16_t port) {
    return std::make_shared<Connection>(std::move(ip), port);
}
