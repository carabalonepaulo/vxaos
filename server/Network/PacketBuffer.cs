using System;

namespace Server.Network
{
    public interface IPacketBuffer
    {
        event Action<byte[]> PacketReceived;
        void Push(byte[] buff);
    }

    public class PacketBuffer : IPacketBuffer
    {
        public event Action<byte[]> PacketReceived;

        CircularBuffer _buffer;
        int _expected;
        int _headerSize;

        public PacketBuffer(int bufferSize)
        {
            _headerSize = sizeof(short);
            _buffer = new CircularBuffer(_headerSize + bufferSize);
            _expected = 0;
        }

        bool TryToRead()
        {
            if (_expected == 0)
            {
                if (!_buffer.CanRead(_headerSize)) return false;

                var lenBuff = _buffer.Read(_headerSize);
                int len = BitConverter.ToInt32(lenBuff, 0);

                _expected = len;
            }

            if (!_buffer.CanRead(_expected)) return false;

            var buff = _buffer.Read(_expected);
            PacketReceived?.Invoke(buff);
            _expected = 0;
            return true;
        }

        public void Push(byte[] buff)
        {
            _buffer.Write(buff);
            while (TryToRead()) ;
        }
    }
}
