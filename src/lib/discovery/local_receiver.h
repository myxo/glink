#pragma once

#include "discovery_service.h"

#include <chrono>
#include <memory>
#include <string>
#include <map>
#include <functional>

namespace net = boost::asio;

class ILocalReceiver {
public:
    virtual std::map<std::string, Endpoint> GetEndpoints() const = 0;

    virtual void OnNewEndpoint(std::function<void(std::string, Endpoint)>) = 0;

    virtual void Stop() = 0;
};

std::shared_ptr<ILocalReceiver> CreateLocalReceiver(net::io_context& io_context);