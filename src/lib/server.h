#pragma once

#include <bits/stdint-uintn.h>
#include <boost/asio.hpp>
#include "boost/asio/ip/tcp.hpp"

namespace net = boost::asio;

class chat_server;

class Server {
public:
    Server(boost::asio::io_context& io_context);
    ~Server();

    void Deliver(std::string msg, std::string_view to_cid);
    std::string GetIp() const;
    uint16_t GetPort() const;
    void MakeConnectionTo(std::string ip, uint16_t port, std::string cid);

private:
    boost::asio::io_context& io_context_;
    std::unique_ptr<chat_server> impl_;
    boost::asio::ip::tcp::endpoint endpoint_;
};
