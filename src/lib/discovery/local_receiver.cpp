#include "local_receiver.h"

#include <boost/asio.hpp>
#include <iostream>
#include <array>
#include <thread>

#include "local_broadcaster.h"

namespace net = boost::asio;

class LocalReceiver : public ILocalReceiver {
public:
    LocalReceiver() : socket_(io_context_) {
        socket_.open(net::ip::udp::v4());
        socket_.bind(net::ip::udp::endpoint(net::ip::address::from_string("192.168.0.101"), kBroadcastPort));
        socket_.set_option(net::socket_base::reuse_address(true));
        socket_.set_option(net::socket_base::broadcast(true));

        // socket_.bind(net::ip::udp::endpoint(net::ip::udp::v4(), kBroadcastPort));
        // socket_.open(net::ip::udp::v4(), kBroadcastPort);
        thread_ = std::jthread{[this] { io_context_.run(); }};
        Receive();
    }
    void Stop() override {}

private:
    void Receive() {
        socket_.async_receive_from(net::buffer(buffer_), remote_endpoint_,
                                   [this](const boost::system::error_code& error, size_t bytes_transferred) {
                                       if (error) {
                                           std::cout << "Receive error: " << error.message();
                                       }
                                       std::cout << "Receive! " << bytes_transferred << " bytes from "
                                                 << remote_endpoint_ << ", " << buffer_.data() << "\n";
                                       Receive();
                                   });
    }

private:
    std::jthread thread_;
    net::io_context io_context_;
    net::ip::udp::socket socket_;
    std::array<char, 1024> buffer_{};
    net::ip::udp::endpoint remote_endpoint_;
};

std::shared_ptr<ILocalReceiver> CreateLocalReceiver() {
    return std::make_shared<LocalReceiver>();
}