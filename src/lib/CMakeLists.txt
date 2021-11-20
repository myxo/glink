add_library(chatik_core STATIC)

set (discovery_sources
	discovery/local_broadcaster.cpp
	discovery/local_broadcaster.h
	discovery/local_receiver.cpp
	discovery/local_receiver.h
	discovery/discovery_service.cpp
)

target_sources(chatik_core PRIVATE
	${discovery_sources}
	server.cpp
	connection.cpp
	connection.h
	utils.cpp
	engine.cpp
	message.h
	include/utils.h
	include/discovery_service.h
	include/engine.h
)

target_include_directories(chatik_core PUBLIC "include")

target_compile_definitions(chatik_core PRIVATE _WIN32_WINNT=0x0601)

# TODO: make more specific choose of public
target_link_libraries(chatik_core PUBLIC ${CONAN_LIBS})

add_subdirectory(tests)
