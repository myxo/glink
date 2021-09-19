add_library(chatik_core STATIC)

target_sources(chatik_core PRIVATE 
	network.cpp
	server.cpp
	client.cpp
	message.h
	include/network.h
)

target_include_directories(chatik_core PUBLIC "include")
