let socket = null;

        function connectWebSocket() {
            socket = new WebSocket("wss://cry9c.life/ws");

            socket.onopen = function() {
                addMessage("Connected to the server");
                fetchImage(); // Fetch image after successful connection
            };

            socket.onmessage = function(event) {
                addMessage("From server: " + event.data);
            };

            socket.onerror = function(error) {
                addMessage("WebSocket error: " + error.toString());
            };

            socket.onclose = function(event) {
                addMessage("Disconnected from the server");
                if (!event.wasClean) {
                    addMessage('Connection closed uncleanly');
                }
            };
        }

        function sendMessage() {
            const message = document.getElementById("messageInput").value;
            if (!socket || socket.readyState !== WebSocket.OPEN) {
                addMessage("Not connected to the server");
                return;
            }
            socket.send(message);
            document.getElementById("messageInput").value = ''; // Clear input after sending
        }

        function addMessage(message) {
            const messagesDiv = document.getElementById("messages");
            messagesDiv.innerHTML += message + '<br>';
            messagesDiv.scrollTop = messagesDiv.scrollHeight; // Auto-scroll to latest message
        }

        // Connect to WebSocket when the page loads
        window.onload = connectWebSocket;