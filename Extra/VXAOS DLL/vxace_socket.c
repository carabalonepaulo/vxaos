//--------------------
#include "vxace_socket.h"
//---------
#include <winsock2.h>
#include <ws2tcpip.h>
#include <stdio.h>
#include <io.h>
//---------------
int SocketLib__GetLastError(void) {
  //----------
  return WSAGetLastError();
  //----------
}
//---------------
int SocketLib__setup(void) {
  //----------
  int err = 0;
  if ((err = WSAStartup(MAKEWORD(2,2), &SocketLib.wsaData)) != 0) {
    WSASetLastError(err);
    return 1;
  }
  //-----
  return 0;
  //----------
}
//---------------
int SocketLib__connect(const char * ip, const char * port, SOCKET * to_socket) {
  //----------
  struct addrinfo * result = NULL, * ptr = NULL, hints;
  //-----
  ZeroMemory( &hints, sizeof(hints) );
  hints.ai_family   = AF_INET;
  hints.ai_socktype = SOCK_STREAM;
  hints.ai_protocol = IPPROTO_TCP;
  //-----
  int err;
  if ((err = getaddrinfo(ip, port, &hints, &result)) != 0) {
    WSASetLastError(err);
    return -1;
  }
  //-----
  SOCKET ConnectSocket = INVALID_SOCKET;
  //-----
  // Try each address until one connects. The previous implementation freed the
  // list inside the loop and kept only the last socket, which could leak or
  // close a good connection.
  for (ptr = result; ptr != NULL; ptr = ptr->ai_next) {
    ConnectSocket = socket(ptr->ai_family, ptr->ai_socktype, ptr->ai_protocol);
    if (ConnectSocket == INVALID_SOCKET) {
      continue;
    }
    if (connect(ConnectSocket, ptr->ai_addr, (int)ptr->ai_addrlen) == 0) {
      break;
    }
    closesocket(ConnectSocket);
    ConnectSocket = INVALID_SOCKET;
  }
  //-----
  freeaddrinfo(result);
  //-----
  if (ConnectSocket == INVALID_SOCKET) {
    return 1;
  }
  //-----
  memcpy(to_socket, &ConnectSocket, sizeof(SOCKET));
  return 0;
  //----------
}
//---------------
int SocketLib__close(SOCKET socket) {
  //----------
  if ((shutdown(socket, SD_SEND)) == SOCKET_ERROR) {
    closesocket(socket);
    return 1;
  }
  //-----
  closesocket(socket);
  return 0;
  //----------
}
//---------------
int SocketLib__send(SOCKET socket, char * datap, int len) {
  //----------
	const uint8_t * data = (const uint8_t *)datap;	/* For pointer arithmetic */
	int sent, left;
  //-----
	/* Keep sending data until it's sent or an error occurs */
	left = len;
	sent = 0;
	WSASetLastError(0);
  //-----
	do {
		len = send(socket, (const char *)data, left, 0);
		if ( len > 0 ) {
			sent += len;
			left -= len;
			data += len;
		}
	} while ( (left > 0) && ((len > 0) || (WSAGetLastError() == WSAEINTR)) );
  //-----
	return sent;
	//----------
}
//---------------
int SocketLib__recv(SOCKET socket, char * data, int maxlen) {
  int total = 0;
  WSASetLastError(0);
  //-----
  while (total < maxlen) {
    int len = recv(socket, (char *)data + total, maxlen - total, 0);
    if (len == SOCKET_ERROR) {
      if (WSAGetLastError() == WSAEINTR) {
        continue;
      }
      break;
    }
    if (len == 0) {
      break;
    }
    total += len;
  }
  //-----
  return total;
}
//---------------
int SocketLib__recv_non_block(SOCKET socket, char * data, int maxlen) {
  //----------
	int len = 0;
  //-----
	WSASetLastError(0);
  //-----
	do {
		len = recv(socket, (char *)data, maxlen, 0);
	} while (WSAGetLastError() == WSAEINTR);
  //-----
	return len;
  //----------
}
//---------------
int SocketLib__eof(SOCKET socket) {
  //----------
  fd_set socks;
  FD_ZERO(&socks);
  FD_SET(socket, &socks);
  //-----
  struct timeval timeout;
  //-----
  timeout.tv_sec  = 0;
  timeout.tv_usec = 0;
  //-----
  return select(socket+1, &socks, (fd_set*)0, (fd_set*)0, &timeout);
  //----------
}
//---------------
int SocketLib__TestHost(const char * ip, const char * port) {
  //----------
  struct addrinfo * result = NULL, hints;
  //-----
  ZeroMemory(&hints, sizeof(hints));
  hints.ai_family   = AF_INET;
  hints.ai_socktype = SOCK_STREAM;
  hints.ai_protocol = IPPROTO_TCP;
  //-----
  int err;
  if ((err = getaddrinfo(ip, port, &hints, &result)) != 0) {
    WSASetLastError(err);
    return 0;
  }
  //-----
  if (result == NULL) {
    return 0;
  } else {
    freeaddrinfo((struct addrinfo*)result);
    return 1;
  }
  //-----
  return 0;
  //----------
}
//--------------------


