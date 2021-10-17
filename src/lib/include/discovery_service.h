#pragma once

#include <boost/asio.hpp>
#include <memory>
#include <string>
#include <map>

namespace net = boost::asio;

constexpr uint16_t kBroadcastPort = 9078;
constexpr const char* kMulticastIp = "239.255.0.1";

struct Endpoint {
    std::string ip;
    uint16_t port;
};

struct BroadcastData {
    std::string id;
    Endpoint ep;
};


class IDiscovery {
public:
    virtual std::map<std::string, Endpoint> GetKnownEndpoints() const = 0;

    virtual void OnNewEndpoint(std::function<void(std::string, Endpoint)>) = 0;

    virtual void SetBroadcastData(BroadcastData data) = 0;
};

std::shared_ptr<IDiscovery> CreateDiscoveryService(net::io_context& io_context);
