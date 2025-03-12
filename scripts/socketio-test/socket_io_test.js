const manager = new io.Manager("http://nogler.ddns.net:8080", {
    transports: ['webtransport'],
});

const socket = manager.socket("/", {
    reconnectionDelayMax: 10000,
    auth: { token: "123" },
    query: {
        "my-key": "my-value"
    }
});

socket.emit("message", { lobbyId: "123" });