#include <spdlog/async.h>
#include <spdlog/sinks/stdout_color_sinks.h>
#include <spdlog/spdlog.h>

#include <boost/uuid/uuid.hpp>
#include <boost/uuid/uuid_generators.hpp>
#include <boost/uuid/uuid_io.hpp>
#include <iostream>
#include <string>

#include "engine.h"

int main(int argc, char* argv[]) {
    spdlog::set_level(spdlog::level::debug);
    auto msg_logger = spdlog::stdout_color_mt("chat_msg");

    std::string uuid = to_string(boost::uuids::random_generator()());
    std::string name = "name_" + uuid;
    if (argc > 1) {
        name = argv[1];
    }
    spdlog::info("My uuid: {}", uuid);

    auto engine = CreateEngine();
    engine->SetSelfInfo(uuid, name);

    while (true)
        ;
}