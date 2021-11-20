#pragma once

#include "message_queue.h"

#include <string>
#include <memory>

#include <boost/asio.hpp>

namespace net = boost::asio;

class IConnection {
public:
    virtual void SendMessage(std::string msg) = 0;
    virtual ~IConnection() = default;
};

class IConnectionPool {
public:
    virtual void AddConnection(std::string id, std::shared_ptr<IConnection> conn) = 0;
    virtual std::shared_ptr<IConnection> GetConnection(std::string_view id) = 0;
};

std::shared_ptr<IConnectionPool> CreateConnectionPool();
std::shared_ptr<IConnection> CreateConnection(std::string ip, uint16_t port, net::io_context& io_context, MessageQueue& mq, std::string from_cid);
std::shared_ptr<IConnection> CreateConnection(net::io_context& io_context, MessageQueue& mq, net::ip::tcp::socket socket, std::string from_cid);
