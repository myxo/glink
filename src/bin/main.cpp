
#include "network.h"
#include <string>

int main(int argc, char *argv[]) {
    if (argc > 1 && std::string_view(argv[1]) == "server") {
        Server server(9078);
        server.Start();
        server.Stop();
    } else {
        auto connection = CreateConnection("127.0.0.1", 9078);
        connection->SendMesasge("Hello world!");
    }
}