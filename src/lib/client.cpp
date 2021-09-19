#include <boost/asio.hpp>
#include <cstdlib>
#include <deque>
#include <iostream>
#include <thread>

#include "message.h"
#include "network.h"

using boost::asio::ip::tcp;

typedef std::deque<chat_message> chat_message_queue;

class chat_client {
public:
    chat_client(boost::asio::io_context& io_context, const tcp::resolver::results_type& endpoints)
        : io_context_(io_context), socket_(io_context) {
        do_connect(endpoints);
    }

    void write(const chat_message& msg) {
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
        boost::asio::async_read(socket_, boost::asio::buffer(read_msg_.data(), chat_message::header_length),
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
                                        std::cout.write(read_msg_.body(), read_msg_.body_length());
                                        std::cout << "\n";
                                        do_read_header();
                                    } else {
                                        socket_.close();
                                    }
                                });
    }

    void do_write() {
        boost::asio::async_write(socket_, boost::asio::buffer(write_msgs_.front().data(), write_msgs_.front().length()),
                                 [this](boost::system::error_code ec, std::size_t /*length*/) {
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
    chat_message read_msg_;
    chat_message_queue write_msgs_;
};

Client::Client(std::string ip, int port) {
    tcp::resolver resolver(io_context_);
    auto endpoints = resolver.resolve(ip.c_str(), std::to_string(port));
    impl_ = std::make_unique<chat_client>(io_context_, endpoints);
}

Client::~Client() = default;

void Client::Send(std::string msg) {
    std::thread t([this]() { io_context_.run(); });

    chat_message cmsg;
    cmsg.body_length(msg.size());
    std::memcpy(cmsg.body(), msg.data(), cmsg.body_length());
    cmsg.encode_header();
    impl_->write(cmsg);
    // impl_->close();
    t.join();
}

// int main(int argc, char* argv[])
//{
//    try
//    {
//        if (argc != 3)
//        {
//            std::cerr << "Usage: chat_client <host> <port>\n";
//            return 1;
//        }
//
//        boost::asio::io_context io_context;
//
//
//

//    }
//    catch (std::exception& e)
//    {
//        std::cerr << "Exception: " << e.what() << "\n";
//    }
//
//    return 0;
//}