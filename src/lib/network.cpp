#include "network.h"
#include "client.h"

#include <boost/asio.hpp>

#include <map>

// TODO: tmp code
class ConnectionPool : public IConnectionPool {
public:
    std::shared_ptr<IConnection> CreateConnection(std::string id, std::string ip, uint16_t port) {
        auto connection = ::CreateConnection(ip, port);
        pool_.insert({id, connection});
        return connection;
    }

    std::map<std::string, std::shared_ptr<IConnection>> pool_;
};

std::shared_ptr<IConnectionPool> CreateConnectionPool() {
    return std::make_shared<ConnectionPool>();
}
