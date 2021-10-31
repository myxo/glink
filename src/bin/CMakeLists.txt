add_executable(chatik_app)

target_sources(chatik_app PRIVATE 
	main.cpp
)

target_link_libraries(chatik_app PRIVATE chatik_core ${CONAN_LIBS})

target_compile_definitions(chatik_app PRIVATE _WIN32_WINNT=0x0601)
