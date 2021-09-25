#include "discovery_service.h"
#include "network.h"
#include "utils.h"

#include <spdlog/spdlog.h>

#include <boost/uuid/uuid.hpp>
#include <boost/uuid/uuid_generators.hpp>
#include <boost/uuid/uuid_io.hpp>
#include <iostream>
#include <string>

int main(int argc, char* argv[]) {
    spdlog::set_level(spdlog::level::debug);

    std::string uuid = to_string(boost::uuids::random_generator()());
    auto connection_pool = CreateConnectionPool();
    auto discovery_service = CreateDiscoveryService();

    discovery_service->OnNewEndpoint([&](std::string id, Endpoint ep) {
        if (id == uuid) {
            return;
        }
        auto connection = connection_pool->CreateConnection(id, ep.ip, ep.port);
        connection->SendMesasge(fmt::format("Hello from {}!", uuid));
    });

    auto local_ips = GetLocalInterfacesIp();

    spdlog::info("My uuid: {}", uuid);

    try {
        Server server;

        discovery_service->SetBroadcastData(
            BroadcastData{.id = uuid, .ep{.ip = local_ips.front(), .port = server.GetPort()}});

        server.Start();

        while (true)
            ;

        server.Stop();

    } catch (std::exception const& e) {
        std::cout << "exception: " << e.what();
    }
}