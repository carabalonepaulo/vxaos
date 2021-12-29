using System;
using Server.Network;
using System.Text.Json;

namespace Server
{
    class Config
    {
        public string Host { get; set; }
        public int Port { get; set; }
        public int Backlog { get; set; }
        public int BufferSize { get; set; }
        public int MaxConnections { get; set; }
    }

    class Program
    {
        static Config _config;
        static Listener _listener;
        static object _listenerEmitter;

        static void Main(string[] args)
        {
            LoadConfig();
            LoadScripts();
            CreateListener();
            Run();
        }

        static void LoadScripts() { }

        static void Run() { }

        static void LoadConfig()
        {
            var text = System.IO.File.ReadAllText("./config.json");
            _config = JsonSerializer.Deserialize<Config>(text);
        }

        static void CreateListenerEmitter()
        {
        }

        static void CreateListener()
        {
            var listener = new Listener(
                _config.Host,
                _config.Port,
                _config.Backlog,
                _config.BufferSize,
                _config.MaxConnections);
            listener.ClientConnected += (cid) => { };
        }
    }
}
