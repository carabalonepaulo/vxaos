using System;
using System.Net;
using System.Net.Sockets;
using System.Collections.Generic;

namespace Server.Network
{
    public interface IListener
    {
        event Action<int> ClientConnected;
        event Action<int, byte[]> DataReceived;
        event Action<int> ClientDisconnected;

        void Start();
        void Stop();
        void Send(int id, byte[] buff);
        void Disconnect(int id);
    }

    public class ClientState
    {
        public int Id { get; set; } = -1;
        public string Ip { get; set; }

        public Socket Socket { get; set; }
        public byte[] Buffer { get; set; }
        public PacketBuffer PacketBuffer { get; set; }
        public bool Disposed { get; set; } = false;

        public Int64 PacketCount { get; set; }
        public Int64 TotalSent { get; set; }
        public Int64 TotalReceived { get; set; }
    }

    public class Listener : IListener
    {
        public event Action<int> ClientConnected;
        public event Action<int, byte[]> DataReceived;
        public event Action<int> ClientDisconnected;

        int _highestIndex = -1;
        Stack<int> _availableIndices;
        ClientState[] _clients;
        Socket _listener;
        string _host;
        int _port;
        int _backlog;
        int _bufferSize;
        bool _disposed;

        public Listener(string host, int port, int backlog, int bufferSize, int maxConnections)
        {
            _host = host;
            _port = port;
            _backlog = backlog;
            _bufferSize = bufferSize;
            _availableIndices = new Stack<int>();
            _clients = new ClientState[maxConnections];
        }

        public void Start()
        {
            _listener = new Socket(AddressFamily.InterNetwork, SocketType.Stream, ProtocolType.Tcp);
            _listener.Bind(new IPEndPoint(IPAddress.Parse(_host), _port));
            _listener.Listen(_backlog);
            _listener.BeginAccept(AcceptCallback, null);
        }

        public void Stop()
        {
            _listener.Close();
            _disposed = true;
            foreach (var client in _clients) if (client != null) Disconnect(client);
        }

        int FreeSlot => _availableIndices.Count > 0 ? _availableIndices.Pop() : _highestIndex++;

        ClientState CreateClientState(Socket socket)
        {
            var client = new ClientState();
            client.Id = FreeSlot;
            client.Ip = ((IPEndPoint)socket.RemoteEndPoint).Address.ToString();
            client.Buffer = new byte[_bufferSize];

            client.PacketBuffer = new PacketBuffer(client.Buffer.Length);
            client.PacketBuffer.PacketReceived += (byte[] buff) =>
            {
                client.PacketCount += 1;
                DataReceived?.Invoke(client.Id, buff);
            };

            client.Socket = socket;
            client.Socket.SendTimeout = 0;
            client.Socket.ReceiveTimeout = 0;
            client.Socket.NoDelay = true;
            client.Socket.BeginReceive(client.Buffer, 0, client.Buffer.Length, SocketFlags.None, ReceiveCallback, client);

            return client;
        }

        void AcceptCallback(IAsyncResult ar)
        {
            if (_disposed) return;
            var client = CreateClientState(_listener.EndAccept(ar));
            _clients[client.Id] = client;
            ClientConnected?.Invoke(client.Id);
            _listener.BeginAccept(AcceptCallback, null);
        }

        void ReceiveCallback(IAsyncResult ar)
        {
            var client = ar.AsyncState as ClientState;
            try
            {
                int length = client.Socket.EndReceive(ar);
                if (length == 0)
                {
                    Disconnect(client);
                    return;
                }

                var buffer = new byte[length];
                Array.Copy(client.Buffer, 0, buffer, 0, length);

                client.TotalSent += length;
                client.PacketBuffer.Push(buffer);

                if (client.Disposed == false)
                    client.Socket.BeginReceive(client.Buffer, 0, client.Buffer.Length, SocketFlags.None, ReceiveCallback, client);
            }
            catch (SocketException) { if (!client.Disposed) Disconnect(client); }
            catch (ObjectDisposedException) { if (!client.Disposed) Disconnect(client); }
        }

        void Disconnect(ClientState client)
        {
            if (client.Disposed) return;

            client.Socket.Close();
            client.Disposed = true;

            _availableIndices.Push(client.Id);
            _clients[client.Id] = null;
            ClientDisconnected?.Invoke(client.Id);

            _listener.BeginAccept(AcceptCallback, null);
        }

        public void Disconnect(int id) => Disconnect(_clients[id]);

        public void Send(int id, byte[] buff)
        {
            var client = _clients[id];
            if (client == null) throw new Exception("Invalid client id.");
            client.Socket.BeginSend(buff, 0, buff.Length, SocketFlags.None, SendCallback, id);
        }

        void SendCallback(IAsyncResult ar)
        {
            var id = (int)ar.AsyncState;
            var client = _clients[id];
            if (client == null || client.Disposed) return;
            client.TotalSent += client.Socket.EndSend(ar);
        }
    }
}