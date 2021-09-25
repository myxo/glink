
#include "network.h"
#include "utils.h"
#include "../../lib/discovery/local_broadcaster.h"
#include "../../lib/discovery/local_receiver.h"

#include <string>
#include <iostream>

#include <spdlog/spdlog.h>

#include <boost/uuid/uuid.hpp>
#include <boost/uuid/uuid_generators.hpp>
#include <boost/uuid/uuid_io.hpp>

int main(int argc, char *argv[]) {
    spdlog::set_level(spdlog::level::debug);

    std::string uuid = to_string(boost::uuids::random_generator()());
    auto connection_pool = CreateConnectionPool();

    spdlog::info("My uuid: {}", uuid);

    try {
        auto receiver = CreateLocalReceiver();
        auto broadcaster = CreateLocalBroadcaster(std::chrono::seconds(1));

        receiver->OnNewEndpoint([&](std::string id, Endpoint ep){
            if (id == uuid) {
                return;
            }
            auto connection = connection_pool->CreateConnection(id, ep.ip, ep.port);
            connection->SendMesasge(fmt::format("Hello from {}!", uuid));
        });


        Server server;
        auto local_ips = GetLocalInterfacesIp();
        
        broadcaster->SetBroadcastData(BroadcastData{.id = uuid, .ip = local_ips.front(), .port = server.GetPort()});
        server.Start();

        while(true);

        server.Stop();

    } catch (std::exception const& e) {
        std::cout << "exception: " << e.what();
    }
}