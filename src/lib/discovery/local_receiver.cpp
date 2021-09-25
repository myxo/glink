#include "local_receiver.h"
#include "local_broadcaster.h"

#include <array>
#include <thread>

#include <boost/asio.hpp>
#include <boost/json.hpp>
#include <spdlog/spdlog.h>
#include <spdlog/fmt/ostr.h>

namespace net = boost::asio;


class LocalReceiver : public ILocalReceiver {
public:
    LocalReceiver() : socket_(io_context_) {
        boost::asio::ip::udp::endpoint listen_endpoint(net::ip::address::from_string("0.0.0.0"), kBroadcastPort);
        socket_.open(listen_endpoint.protocol());
        socket_.set_option(boost::asio::ip::udp::socket::reuse_address(true));
        socket_.bind(listen_endpoint);

        socket_.set_option(boost::asio::ip::multicast::join_group(net::ip::address::from_string(kMulticastIp)));

        thread_ = std::jthread{[this] { io_context_.run(); }};
        Receive();
    }
    void Stop() override {}

    std::map<std::string, Endpoint> GetEndpoints() const override {
        return nodes_;
    }

    void OnNewEndpoint(std::function<void(std::string, Endpoint)> callback) {
        callback_ = callback;
    }

private:
    void Receive() {
        socket_.async_receive_from(net::buffer(buffer_), remote_endpoint_,
                                   [this](const boost::system::error_code& error, size_t bytes_transferred) {
                                       if (error) {
                                           spdlog::warn("Receive error: {}", error.message());
                                       }
                                       // spdlog::trace("LocalReceiver: receive {} bytes from {}. '{}'", bytes_transferred, remote_endpoint_, buffer_.data());

                                        boost::json::value val = boost::json::parse(buffer_.data());
                                        Endpoint ep;
                                        ep.ip = boost::json::value_to<std::string>(val.as_object()["ip"]);
                                        ep.port = boost::json::value_to<uint16_t>(val.as_object()["port"]);
                                        std::string id = boost::json::value_to<std::string>(val.as_object()["id"]);
                                            
                                        auto [_, new_elem] = nodes_.insert({id, ep});
                                        if (new_elem) {
                                            spdlog::info("Receiver: get new node. Id - {}, endpoint - {}:{}", id, ep.ip, ep.port);
                                            callback_(id, ep);
                                        }

                                       Receive();
                                   });
    }

private:
    std::jthread thread_;
    net::io_context io_context_;
    net::ip::udp::socket socket_;
    std::array<char, 1024> buffer_{};
    net::ip::udp::endpoint remote_endpoint_;

    std::map<std::string, Endpoint> nodes_;
    std::function<void(std::string, Endpoint)> callback_;
};

std::shared_ptr<ILocalReceiver> CreateLocalReceiver() {
    return std::make_shared<LocalReceiver>();
}