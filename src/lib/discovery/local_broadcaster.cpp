#include "local_broadcaster.h"

#include <boost/asio.hpp>
#include <thread>
#include <iostream>
#include <boost/json.hpp>

// Some boost hack. TODO: make special cpp file for this
#include <boost/json/src.hpp>

namespace net = boost::asio;

class LocalBroadcaster : public ILocalBroadcaster {
public:
    LocalBroadcaster(std::chrono::milliseconds period)
        : socket_(io_context_, net::ip::udp::endpoint(net::ip::address::from_string("192.168.0.101"), 0)),
          endpoint_(net::ip::address_v4::broadcast(), kBroadcastPort),
          timer_(io_context_, period),
          period_ (period){

        //socket_.open(net::ip::udp::v4());
        //socket_.set_option(net::ip::udp::socket::reuse_address(true));
        socket_.set_option(net::socket_base::broadcast(true));
        thread_ = std::jthread{[this] { io_context_.run(); }};
        ScheduleTimer();
    }

    ~LocalBroadcaster() {
        socket_.close();
    }

    void SetBroadcastData(BroadcastData data) override {
        // TODO: some binary representation
        boost::json::value val = {{"id", data.id}, {"ip", data.ip}, {"port", data.port}};

        auto val_str = boost::json::serialize(val);
        data_ = std::move(val_str);
    }

    void Stop() override {}

private:
    void ScheduleTimer() {
        timer_.expires_after(period_);
        timer_.async_wait([this](const boost::system::error_code& e) {
            if (!e) {
                SendData();
                ScheduleTimer();
            }
        });
    }

    void SendData() {
        if (data_.empty())
            return;

        static int index = 0;
        std::cout << index++ << ". Broadcast send\n";
        socket_.send_to(net::buffer(data_), endpoint_);
    }

private:
    // TODO: mutex
    std::string data_;

    std::jthread thread_;
    net::io_context io_context_;
    net::ip::udp::socket socket_;
    net::ip::udp::endpoint endpoint_;
    net::steady_timer timer_;
    std::chrono::milliseconds period_;
};

std::shared_ptr<ILocalBroadcaster> CreateLocalBroadcaster(std::chrono::milliseconds period) {
    return std::make_shared<LocalBroadcaster>(period);
}