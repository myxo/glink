#include "connection.h"
#include "message.h"
#include "message_queue.h"
#include "utils.h"

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

class chat_client : public std::enable_shared_from_this<chat_client> {
public:
    chat_client(boost::asio::io_context& io_context, MessageQueue& mq, const tcp::resolver::results_type& endpoints, std::string cid)
        : io_context_(io_context)
        , socket_(io_context)
        , timer_(socket_.get_executor()) 
        , mq_(mq)
        , own_cid_(cid)
    {
        spdlog::debug("chat_client ctor with endpoints");
        co_spawn(io_context_, [this, eps = endpoints] { 
                return Connect(eps); 
        }, net::detached);
    }

    chat_client(net::io_context& io_context, MessageQueue& mq, net::ip::tcp::socket socket, std::string cid)
        : io_context_(io_context)
          , socket_(std::move(socket))
          , timer_(socket_.get_executor()) 
          , mq_(mq)
          , own_cid_(cid)
    {
        spdlog::debug("chat_client ctor with socket");
        co_spawn(socket_.get_executor(), [this] { 
                return Reader(); 
        }, net::detached);
    }

    void write(const MessagesReply& msg) {
        boost::asio::post(io_context_, [this_ = shared_from_this(), msg = msg]() mutable {
            bool write_in_progress = !this_->write_msgs_.empty();
            this_->write_msgs_.push_back(std::move(msg));
            if (!write_in_progress) {
                this_->timer_.cancel_one();
            }
        });
    }

    void close() {
        // TODO: wait until all handlers is executed?
        spdlog::info("Close socket: {}", GetEndpointAsStr(socket_));
        boost::asio::post(io_context_, [this_ = shared_from_this()]() { 
            if (this_->socket_.is_open())
                this_->socket_.close(); 
        });
        timer_.cancel_one();
    }

private:
    net::awaitable<void> Connect(const tcp::resolver::results_type& endpoints) {
        co_await net::async_connect(socket_, endpoints, net::use_awaitable);

        // Handshake
        Packet read_msg;
        std::size_t length = co_await net::async_read(
            socket_, boost::asio::buffer(read_msg.GetHeaderAsString(), Packet::kHeaderSize), net::use_awaitable);

        if (!read_msg.decode_header()) {
            spdlog::info("[MakeConnectionTo] error on reading header, cannot decode. Endpoint {}", 
                    GetEndpointAsStr(socket_));
            co_return;
        }
        spdlog::trace("[MakeConnectionTo] read header ({} bytes), now async read body ({} bytes)", length,
                      read_msg.GetBodySize());

        std::string payload;
        payload.resize(read_msg.GetBodySize());
        length = co_await net::async_read(socket_, net::buffer(payload, read_msg.GetBodySize()),
                                          net::use_awaitable);

        // TODO: check type enum
        try {
            auto reply = DeserializePacket<UserMetaRequest>(payload);
            // spdlog::debug("get meta. cid: {}, name: {}", reply.client_cid, reply.name);
        } catch (cereal::Exception const& e) { 
            // TODO: do not leak cereal
            // TODO: do not use warn?
            spdlog::warn("[MakeConnectionTo]: error while parse of UserMetaReply: {}\nPayload is {}", e.what(), payload);
            co_return;
        }

        Packet send_pkg(UserMetaReply{own_cid_, "test_name"});
        // Packet send_pkg(UserMetaReply{cid_, "test_name"});
        co_await net::async_write(socket_,
                                  net::buffer(send_pkg.GetHeaderAsString(), Packet::kHeaderSize),
                                  net::use_awaitable);
        co_await net::async_write(
            socket_, net::buffer(send_pkg.GetPayload(), send_pkg.GetBodySize()), net::use_awaitable);
        
        // Start reading and writing actual messages
        co_spawn(
            socket_.get_executor(), [this_ = shared_from_this()] { return this_->Reader(); }, net::detached);

        co_spawn(
            socket_.get_executor(), [this_ = shared_from_this()] { return this_->Writer(); }, net::detached);

        write(MessagesReply{"hello from " + own_cid_});
    }

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

            spdlog::debug("Client: finish read body ({} bytes): {}", length, read_msg.GetBodySize());
            PutInMq(mq_, read_msg.GetHeader(), payload);
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
    MessageQueue& mq_;
    chat_message_queue write_msgs_;
    std::string own_cid_;
};

class Connection : public IConnection {
public:
    Connection(std::string ip, uint16_t port, net::io_context& io_context, MessageQueue& mq, std::string from_cid) 
        : io_context_(io_context)
          , mq_(mq)
    {
        tcp::resolver resolver(io_context_);
        auto endpoints = resolver.resolve(ip.c_str(), std::to_string(port));
        client_ = std::make_shared<chat_client>(io_context_, mq_, endpoints, from_cid);
        spdlog::debug("Create new connection to {}:{}", ip, port);
    }

    Connection(net::io_context& io_context, MessageQueue& mq, net::ip::tcp::socket socket, std::string from_cid) 
        : io_context_(io_context)
          , mq_(mq)
    {
        spdlog::debug("Create from connection to {}:{}", socket.remote_endpoint().address().to_string(), 
                socket.remote_endpoint().port());
        client_ = std::make_shared<chat_client>(io_context_, mq_, std::move(socket), from_cid);
    }

    ~Connection() {
        spdlog::debug("Close connection");
        client_->close();
    }

    void SendMessage(std::string msg) override {
        client_->write(MessagesReply{msg});
    }

private:
    net::io_context& io_context_;
    std::shared_ptr<chat_client> client_;
    MessageQueue& mq_;
};

std::shared_ptr<IConnection> CreateConnection(std::string ip, uint16_t port, net::io_context& io_context, MessageQueue& mq, std::string from_cid) {
    return std::make_shared<Connection>(std::move(ip), port, io_context, mq, from_cid);
}

std::shared_ptr<IConnection> CreateConnection(net::io_context& io_context, MessageQueue& mq, net::ip::tcp::socket socket, std::string from_cid) {
    return std::make_shared<Connection>(io_context, mq, std::move(socket), from_cid);
}
