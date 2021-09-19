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

class chat_client;
class Client {
public:
    Client(std::string ip, int port);
    ~Client();
    void Send(std::string msg);

private:
    boost::asio::io_context io_context_;
    std::unique_ptr<chat_client> impl_;
};