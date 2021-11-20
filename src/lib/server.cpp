#include "connection.h"
#include "message.h"
#include "message_queue.h"
#include "server.h"
#include "utils.h"

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
    chat_server(boost::asio::io_context& io_context, MessageQueue& mq, const tcp::endpoint& endpoint, std::string cid) 
        : io_context_(io_context)
        , acceptor_(io_context, endpoint)
        , cid_(cid)
        , mq_(mq)
    {
        connection_pool_ = CreateConnectionPool();

        co_spawn(io_context_, [this] { return Acceptor(); }, net::detached);
    }

public:
    net::awaitable<void> Acceptor() {
        while (true) {
            try {
                tcp::socket socket = co_await acceptor_.async_accept(net::use_awaitable);
                spdlog::debug("Server: accept new connection from {}:{}", socket.remote_endpoint().address().to_string(),
                              socket.remote_endpoint().port());

                Packet send_pkg(UserMetaRequest{});
                co_await net::async_write(socket,
                                          net::buffer(send_pkg.GetHeaderAsString(), Packet::kHeaderSize),
                                          net::use_awaitable);
                co_await net::async_write(
                    socket, net::buffer(send_pkg.GetPayload(), send_pkg.GetBodySize()), net::use_awaitable);

                Packet read_msg;
                std::size_t length = co_await net::async_read(
                    socket, boost::asio::buffer(read_msg.GetHeaderAsString(), Packet::kHeaderSize), net::use_awaitable);

                if (!read_msg.decode_header()) {
                    spdlog::info("Acceptor: error on reading header, cannot decode. Endpoint {}", GetEndpointAsStr(socket));
                    continue;
                }
                spdlog::trace("Acceptor: read header ({} bytes), now async read body ({} bytes)", length,
                              read_msg.GetBodySize());

                std::string payload;
                payload.resize(read_msg.GetBodySize());
                length = co_await net::async_read(socket, net::buffer(payload, read_msg.GetBodySize()),
                                                  net::use_awaitable);

                try {
                    auto reply = DeserializePacket<UserMetaReply>(payload);
                    spdlog::debug("get meta. cid: {}, name: {}", reply.client_cid, reply.name);

                    connection_pool_->AddConnection(reply.client_cid, CreateConnection(io_context_, mq_, std::move(socket), cid_));
                } catch (cereal::Exception const& e) { 
                    // TODO: do not leak cereal
                    // TODO: do not use warn?
                    spdlog::warn("[Accpetor]: error while parse of UserMetaReply: {}\nPayload is {}", e.what(), payload);
                }
            } catch (std::exception& e) {
                spdlog::warn("Exception in Acceptor: {}", e.what());
            }
        }
    }

    net::io_context& io_context_;
    tcp::acceptor acceptor_;
    std::shared_ptr<IConnectionPool> connection_pool_;
    // TODO: get struct for own identity (?)
    std::string cid_;
    MessageQueue& mq_;
};

Server::Server(boost::asio::io_context& io_context, MessageQueue& mq, std::string cid) 
    : io_context_(io_context)
      , endpoint_(tcp::v4(), 0) 
      , own_cid_(cid)
      , mq_(mq)
{
    impl_ = std::make_unique<chat_server>(io_context_, mq_, endpoint_, cid);
    spdlog::info("Create server on {}:{}", GetIp(), GetPort());
}

Server::~Server() = default;

void Server::Deliver(std::string msg, std::string_view to_cid) {
    auto conn = impl_->connection_pool_->GetConnection(to_cid);
    conn->SendMessage(msg);
}

void Server::MakeConnectionTo(std::string ip, uint16_t port, std::string cid) {
    impl_->connection_pool_->AddConnection(cid, CreateConnection(ip, port, io_context_, mq_, own_cid_));
}

std::string Server::GetIp() const {
    return impl_->acceptor_.local_endpoint().address().to_string();
}

uint16_t Server::GetPort() const {
    return impl_->acceptor_.local_endpoint().port();
}
