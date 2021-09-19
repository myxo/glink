add_library(chatik_core STATIC)

set (discovery_sources
	discovery/local_broadcaster.cpp
	discovery/local_broadcaster.h
	discovery/local_receiver.cpp
	discovery/local_receiver.h
)

target_sources(chatik_core PRIVATE
	${discovery_sources}
	network.cpp
	server.cpp
	client.cpp
	message.h
	include/network.h
)

target_include_directories(chatik_core PUBLIC "include")

target_compile_definitions(chatik_core PRIVATE _WIN32_WINNT=0x0601)

target_link_libraries(chatik_core PRIVATE ${CONAN_LIBS} )
