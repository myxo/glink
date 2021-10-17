#include "local_broadcaster.h"

#include <boost/asio.hpp>
#include <thread>
#include <boost/json.hpp>

// Some boost hack. TODO: make special cpp file for this
#include <boost/json/src.hpp>

namespace net = boost::asio;

class LocalBroadcaster : public ILocalBroadcaster {
public:
    LocalBroadcaster(std::chrono::milliseconds period, net::io_context& io_context)
        : io_context_(io_context),
          socket_(io_context_, endpoint_.protocol()),
          endpoint_(net::ip::address::from_string(kMulticastIp), kBroadcastPort),
          timer_(io_context_, period),
          period_ (period){

        ScheduleTimer();
    }

    ~LocalBroadcaster() {
        socket_.close();
    }

    void SetBroadcastData(BroadcastData data) override {
        // TODO: some binary representation
        boost::json::value val = {{"id", data.id}, {"ip", data.ep.ip}, {"port", data.ep.port}};

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
        socket_.send_to(net::buffer(data_), endpoint_);
    }

private:
    // TODO: mutex
    std::string data_;

    net::io_context& io_context_;
    net::ip::udp::endpoint endpoint_;
    net::ip::udp::socket socket_;
    net::steady_timer timer_;
    std::chrono::milliseconds period_;
};

std::shared_ptr<ILocalBroadcaster> CreateLocalBroadcaster(std::chrono::milliseconds period,
                                                          net::io_context& io_context) {
    return std::make_shared<LocalBroadcaster>(period, io_context);
}