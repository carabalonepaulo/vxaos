using System;

namespace Server.Network
{
    public class CircularBuffer
    {
        byte[] buffer;
        int writerPosition;
        int readerPosition;
        int availableToWrite;
        int availableToRead;

        public int Length => buffer.Length;

        public bool CanRead(int len) => availableToRead >= len;
        public bool CanWrite(int len) => availableToWrite >= len;

        public CircularBuffer(int bufferSize = 1024)
        {
            buffer = new byte[bufferSize];
            writerPosition = 0;
            readerPosition = 0;
            availableToWrite = bufferSize;
            availableToRead = 0;
        }

        void AdvanceWriter(int len)
        {
            writerPosition += len;
            availableToRead += len;
            availableToWrite -= len;
        }

        void AdvanceReader(int len)
        {
            readerPosition += len;
            availableToRead -= len;
            availableToWrite += len;
        }

        public void Write(byte[] buff)
        {
            var len = buff.Length;
            if (!CanWrite(len)) throw new Exception("No space availble to write");

            if (writerPosition + len >= buffer.Length)
            {
                var firstChunkLength = buffer.Length - writerPosition;
                var secondChunkLength = buff.Length - firstChunkLength;

                Array.Copy(buff, 0, buffer, writerPosition, firstChunkLength);
                AdvanceWriter(firstChunkLength);
                writerPosition = 0;

                Array.Copy(buff, firstChunkLength, buffer, writerPosition, secondChunkLength);
                AdvanceWriter(secondChunkLength);

                return;
            }

            Array.Copy(buff, 0, buffer, writerPosition, len);
            AdvanceWriter(len);
        }

        public byte[] Read(int len)
        {
            if (!CanRead(len)) throw new Exception("No data to read.");

            if (readerPosition + len >= buffer.Length)
            {
                var firstChunkLen = buffer.Length - readerPosition;
                var secondChunkLen = len - firstChunkLen;
                var buff = new byte[len];

                Array.Copy(buffer, readerPosition, buff, 0, firstChunkLen);
                AdvanceReader(firstChunkLen);
                readerPosition = 0;
                Array.Copy(buffer, readerPosition, buff, firstChunkLen, secondChunkLen);
                AdvanceReader(secondChunkLen);

                return buff;
            }

            var resultBuffer = new byte[len];
            Array.Copy(buffer, readerPosition, resultBuffer, 0, len);
            AdvanceReader(len);

            return resultBuffer;
        }
    }
}
