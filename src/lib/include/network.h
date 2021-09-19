#pragma once

#include <string>
#include <memory>

#include <boost/asio.hpp>

class chat_server;

class Server {
public:
    Server(int port);
    ~Server();
    void Start();
    void Stop();

private:
    boost::asio::io_context io_context_;
    std::unique_ptr<chat_server> impl_;
};

class IConnection {
public:
    virtual void SendMesasge(std::string msg) = 0;
};
std::shared_ptr<IConnection> CreateConnection(std::string ip, uint16_t port);