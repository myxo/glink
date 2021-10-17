#include "network.h"
#include "client.h"

#include <boost/asio.hpp>

#include <map>

// TODO: tmp code
class ConnectionPool : public IConnectionPool {
public:

    void AddConnection(std::string id, std::shared_ptr<IConnection> conn) {
        pool_.insert({id, conn});
    }

private:
    std::map<std::string, std::shared_ptr<IConnection>> pool_;
};

std::shared_ptr<IConnectionPool> CreateConnectionPool() {
    return std::make_shared<ConnectionPool>();
}
