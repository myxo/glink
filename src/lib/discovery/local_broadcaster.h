#pragma once

#include "discovery_service.h"

#include <memory>
#include <string>
#include <chrono>

namespace net = boost::asio;

class ILocalBroadcaster {
public:
    virtual void SetBroadcastData(BroadcastData data) = 0;
    virtual void Stop() = 0;
};

std::shared_ptr<ILocalBroadcaster> CreateLocalBroadcaster(std::chrono::milliseconds period,
                                                          net::io_context& io_context);