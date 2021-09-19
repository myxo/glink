
#include "network.h"
#include "../../lib/discovery/local_broadcaster.h"
#include "../../lib/discovery/local_receiver.h"

#include <string>
#include <iostream>

#include <spdlog/spdlog.h>

int main(int argc, char *argv[]) {
    spdlog::set_level(spdlog::level::trace);
    try {
        auto receiver = CreateLocalReceiver();
        auto broadcaster = CreateLocalBroadcaster(std::chrono::seconds(1));
        broadcaster->SetBroadcastData(BroadcastData{.id = "test_id", .ip = "192.168.0.101", .port = 1234});

        while(true);

    } catch (std::exception const& e) {
        std::cout << "exception: " << e.what();
    }


    return 0;
    if (argc > 1 && std::string_view(argv[1]) == "server") {
        Server server(9078);
        server.Start();
        server.Stop();
    } else {
        auto connection = CreateConnection("127.0.0.1", 9078);
        connection->SendMesasge("Hello world!");
    }
}