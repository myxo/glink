#include "connection.h"
#include "message.h"

#include <spdlog/spdlog.h>
#include <boost/asio.hpp>

#include <cstdlib>
#include <deque>
#include <iostream>
#include <thread>
#include <optional>


using boost::asio::ip::tcp;

namespace net = boost::asio;

typedef std::deque<Packet> chat_message_queue;

class ConnectionPool : public IConnectionPool {
public:

    void AddConnection(std::string id, std::shared_ptr<IConnection> conn) override {
        pool_.insert({id, conn});
    }

    std::shared_ptr<IConnection> GetConnection(std::string_view id) override {
        if (auto it = pool_.find(id); it != pool_.end()) {
            return it->second;
        }
        return {};
    }

private:
    std::map<std::string, std::shared_ptr<IConnection>, std::less<>> pool_;
};

std::shared_ptr<IConnectionPool> CreateConnectionPool() {
    return std::make_shared<ConnectionPool>();
}

class chat_client {
public:
    chat_client(boost::asio::io_context& io_context, const tcp::resolver::results_type& endpoints)
        : io_context_(io_context), socket_(io_context), timer_(socket_.get_executor()) {
        do_connect(endpoints);
    }

    chat_client(net::io_context& io_context, net::ip::tcp::socket socket)
        : io_context_(io_context), socket_(std::move(socket)), timer_(socket_.get_executor()) {
        co_spawn(socket_.get_executor(), [this] { return Reader(); }, net::detached);
    }

    void write(const MessagesReply& msg) {
        boost::asio::post(io_context_, [this, msg]() mutable {
            bool write_in_progress = !write_msgs_.empty();
            write_msgs_.push_back(std::move(msg));
            if (!write_in_progress) {
                timer_.cancel_one();
            }
        });
    }

    void close() {
        boost::asio::post(io_context_, [this]() { socket_.close(); });
        timer_.cancel_one();
    }

private:
    void do_connect(const tcp::resolver::results_type& endpoints) {
        boost::asio::async_connect(socket_, endpoints, [this](boost::system::error_code ec, tcp::endpoint) {
            if (!ec) {
                co_spawn(
                    socket_.get_executor(), [this] { return Reader(); }, net::detached);

                co_spawn(
                    socket_.get_executor(), [this] { return Writer(); }, net::detached);
            }
        });
    }

    // void do_read_header() {
    //    boost::asio::async_read(socket_, boost::asio::buffer(read_msg_.data(), Packet::kHeaderSize),
    //                            [this](boost::system::error_code ec, std::size_t /*length*/) {
    //                                if (!ec && read_msg_.decode_header()) {
    //                                    do_read_body();
    //                                } else {
    //                                    socket_.close();
    //                                }
    //                            });
    //}

    // void do_read_body() {
    //    boost::asio::async_read(socket_, boost::asio::buffer(read_msg_.body(), read_msg_.body_length()),
    //                            [this](boost::system::error_code ec, std::size_t /*length*/) {
    //                                if (!ec) {
    //                                    //std::cout.write(read_msg_.body(), read_msg_.body_length());
    //                                    //std::cout << "\n";
    //                                    do_read_header();
    //                                } else {
    //                                    socket_.close();
    //                                }
    //                            });
    //}

    // void do_write() {
    //    boost::asio::async_write(socket_,
    //                             boost::asio::buffer(write_msgs_.front().data(), write_msgs_.front().get_length()),
    //                             [this](boost::system::error_code ec, std::size_t length) {
    //                                spdlog::trace("Connection: write complete, wrote {} bytes", length);
    //                                 if (!ec) {
    //                                     write_msgs_.pop_front();
    //                                     if (!write_msgs_.empty()) {
    //                                         do_write();
    //                                     }
    //                                 } else {
    //                                     socket_.close();
    //                                 }
    //                             });
    //}

    net::awaitable<void> Reader() try {
        while (true) {
            Packet read_msg;
            std::size_t length = co_await net::async_read(
                socket_, boost::asio::buffer(read_msg.GetHeaderAsString(), Packet::kHeaderSize), net::use_awaitable);

            if (!read_msg.decode_header()) {
                spdlog::info("Client: error on reading header, cannot decode");
                co_return;
            }
            spdlog::trace("Client: read header ({} bytes), now async read body ({} bytes)", length,
                          read_msg.GetBodySize());

            std::string payload;
            payload.resize(read_msg.GetBodySize());
            length = co_await net::async_read(socket_, net::buffer(payload, read_msg.GetBodySize()),
                                              net::use_awaitable);

            read_msg.GetPayloadMut() = payload;

            spdlog::debug("Client: finish read body ({} bytes): {}", length, read_msg.GetBodySize());
            spdlog::get("chat_msg")->info("{}", read_msg.GetPayload());
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
                                          net::buffer(write_msgs_.front().GetHeaderAsString(), Packet::kHeaderSize),
                                          net::use_awaitable);
                co_await net::async_write(
                    socket_, net::buffer(write_msgs_.front().GetPayload(), write_msgs_.front().GetBodySize()),
                    net::use_awaitable);
                write_msgs_.pop_front();
            }
        }

    } catch (std::exception& e) {
        spdlog::warn("Exception in Writer: {}", e.what());
    }

private:
    net::io_context& io_context_;
    tcp::socket socket_;
    Packet read_msg_;
    net::steady_timer timer_;
    chat_message_queue write_msgs_;
};

class Connection : public IConnection {
public:
    Connection(std::string ip, uint16_t port, net::io_context& io_context) : io_context_(io_context) {
        tcp::resolver resolver(io_context_);
        auto endpoints = resolver.resolve(ip.c_str(), std::to_string(port));
        client_.emplace(io_context_, endpoints);
        spdlog::debug("Create new connection to {}:{}", ip, port);
    }

    Connection(net::io_context& io_context, net::ip::tcp::socket socket) 
        : io_context_(io_context)
    {
        spdlog::debug("Create from connection to {}:{}", socket.remote_endpoint().address().to_string(), 
                socket.remote_endpoint().port());
        client_.emplace(io_context_, std::move(socket));
    }

    ~Connection() {
        client_->close();
    }

    void SendMessage(std::string msg) override {
        client_->write(MessagesReply{msg});
    }

private:
    net::io_context& io_context_;
    std::optional<chat_client> client_;
};

std::shared_ptr<IConnection> CreateConnection(std::string ip, uint16_t port, net::io_context& io_context) {
    return std::make_shared<Connection>(std::move(ip), port, io_context);
}

std::shared_ptr<IConnection> CreateConnection(net::io_context& io_context, net::ip::tcp::socket socket) {
    return std::make_shared<Connection>(io_context, std::move(socket));
}
