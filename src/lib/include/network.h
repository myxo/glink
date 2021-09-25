#pragma once

#include <string>
#include <memory>

#include <boost/asio.hpp>

class chat_server;

class Server {
public:
    Server();
    ~Server();
    void Start();
    void Stop();
    std::string GetIp() const;
    uint16_t GetPort() const;

private:
    boost::asio::io_context io_context_;
    std::unique_ptr<chat_server> impl_;
    boost::asio::ip::tcp::endpoint endpoint_;
};

class IConnection {
public:
    virtual void SendMesasge(std::string msg) = 0;
};

class IConnectionPool {
public:
    virtual std::shared_ptr<IConnection> CreateConnection(std::string id, std::string ip, uint16_t port) = 0;
};

std::shared_ptr<IConnectionPool> CreateConnectionPool();
