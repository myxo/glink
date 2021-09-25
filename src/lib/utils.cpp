#include "utils.h"

#include <boost/asio.hpp>
#include <spdlog/spdlog.h>

std::vector<std::string> GetLocalInterfacesIp() {
    using boost::asio::ip::tcp;
    boost::asio::io_service io_service;
    std::vector<std::string> result;

    tcp::resolver resolver(io_service);
    tcp::resolver::query query(boost::asio::ip::host_name(), "");
    tcp::resolver::iterator it = resolver.resolve(query);

    while (it != tcp::resolver::iterator()) {
        boost::asio::ip::address addr = (it++)->endpoint().address();
        if (addr.is_v6()) {
            spdlog::debug("Found ipv6 ip: {}", addr.to_string());
        } else {
            result.push_back(addr.to_string());
        }
    }
    return result;
}