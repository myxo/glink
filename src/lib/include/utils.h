#pragma once

#include <string>
#include <vector>

#include <boost/asio.hpp>
#include <fmt/format.h>

namespace net = boost::asio;

std::vector<std::string> GetLocalInterfacesIp();

static std::string GetEndpointAsStr(net::ip::tcp::socket const& socket) {
    return fmt::format("{}:{}", socket.remote_endpoint().address().to_string(),
                          socket.remote_endpoint().port());
} 
