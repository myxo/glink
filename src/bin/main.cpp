#include <spdlog/async.h>
#include <spdlog/sinks/stdout_color_sinks.h>
#include <spdlog/spdlog.h>

#include <boost/uuid/uuid.hpp>
#include <boost/uuid/uuid_generators.hpp>
#include <boost/uuid/uuid_io.hpp>
#include <boost/asio.hpp>
#include <chrono>
#include <iostream>
#include <string>

#include "boost/asio/io_context.hpp"
#include "engine.h"

boost::asio::io_context io_context;

void handler(const boost::system::error_code& error, int signal_number) {
  if (!error) {
      io_context.stop();
  }
}

int main(int argc, char* argv[]) {
    spdlog::set_level(spdlog::level::debug);
    auto msg_logger = spdlog::stdout_color_mt("chat_msg");


    boost::asio::signal_set signals(io_context, SIGINT, SIGTERM);
    signals.async_wait(handler);

    std::string uuid = to_string(boost::uuids::random_generator()());
    std::string name = "name_" + uuid;
    if (argc > 1) {
        name = argv[1];
    }
    spdlog::info("My uuid: {}", uuid);

    auto engine = CreateEngine(io_context);
    engine->SetSelfInfo(uuid, name);

    io_context.run();

}
