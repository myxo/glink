#pragma once

#include <string>
#include <memory>

#include <boost/asio.hpp>

class chat_server;

class Server {
public:
    Server(boost::asio::io_context& io_context);
    ~Server();
    std::string GetIp() const;
    uint16_t GetPort() const;

private:
    boost::asio::io_context& io_context_;
    std::unique_ptr<chat_server> impl_;
    boost::asio::ip::tcp::endpoint endpoint_;
};

class IConnection {
public:
    virtual void SendMesasge(std::string msg) = 0;
};

class IConnectionPool {
public:
    virtual void AddConnection(std::string id, std::shared_ptr<IConnection> conn) = 0;
};

std::shared_ptr<IConnectionPool> CreateConnectionPool();
