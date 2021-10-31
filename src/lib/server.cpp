#include "connection.h"
#include "message.h"
#include "server.h"

#include <boost/asio.hpp>
#include <spdlog/spdlog.h>

#include <cstdlib>
#include <deque>
#include <list>
#include <memory>
#include <set>
#include <utility>


using boost::asio::ip::tcp;

namespace net = boost::asio;

//----------------------------------------------------------------------

typedef std::deque<Packet> chat_message_queue;


class chat_server {
public:
    chat_server(boost::asio::io_context& io_context, const tcp::endpoint& endpoint) 
        : io_context_(io_context)
        , acceptor_(io_context, endpoint) 
    {
        connection_pool_ = CreateConnectionPool();
        do_accept();
    }

public:
    void do_accept() {
        acceptor_.async_accept([this](boost::system::error_code ec, tcp::socket socket) {
            spdlog::debug("Server: accept new connection from {}:{}", socket.remote_endpoint().address().to_string(),
                          socket.remote_endpoint().port());
            if (!ec) {
                connection_pool_->AddConnection("", CreateConnection(io_context_, std::move(socket)));
            }

            do_accept();
        });
    }

    net::io_context& io_context_;
    tcp::acceptor acceptor_;
    std::shared_ptr<IConnectionPool> connection_pool_;
};

Server::Server(boost::asio::io_context& io_context) : io_context_(io_context), endpoint_(tcp::v4(), 0) {
    impl_ = std::make_unique<chat_server>(io_context_, endpoint_);
    spdlog::info("Create server on {}:{}", GetIp(), GetPort());
}

Server::~Server() = default;

void Server::Deliver(std::string msg, std::string_view to_cid) {
    auto conn = impl_->connection_pool_->GetConnection(to_cid);
    conn->SendMessage(msg);
}

void Server::MakeConnectionTo(std::string ip, uint16_t port, std::string cid) {
    impl_->connection_pool_->AddConnection(cid, CreateConnection(ip, port, io_context_));
}

std::string Server::GetIp() const {
    return impl_->acceptor_.local_endpoint().address().to_string();
}

uint16_t Server::GetPort() const {
    return impl_->acceptor_.local_endpoint().port();
}
