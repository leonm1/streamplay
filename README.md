# Streamplay

Streamplay is a program designed to facilitate automatic discovery and streaming of an audio (or video) source from one computer to another, using FFmpeg as a backend.

## Server

The server uses zeroconf to browse the LAN for a client broadcasting as "_streamplay-client._tcp". Once a client is discovered, the server logs the client's IP in a list of active streams and uses an os/exec call to start an rtp stream to the client using FFmpeg.

Usage:

```usage
$ server.exe [OPTIONS]... -a "Audio device" [-v "Video device"]

-dev
    Lists available input devices
-a string
    Audio device for stream (Select from output of -dev flag)
-v string
    Video device for stream (Select from output of -dev flag)
-d
    Output debug information
-h
    Output this usage information
```

## Client

The client broadcasts as a zeroconf service "_streamplay-client._tcp" on the local LAN, then opens ffplay to recieve an rtp stream on the system's preferred IP address.

Usage:

```usage
$ client.exe [OPTIONS]

-dev
    Lists available network interfaces
-iface string
    Network interface on which to listen
-d
    Output debug information
-h
    Output this usage information
```

## Background

This program was originally designed to provide set-and-forget streaming of a live event over wifi for my church. It streamed from a computer attached to the sound system to a raspberry pi in the nursery. Priorities for this project were for the server to begin only one stream to each client at a time and for a client to handle a crashed server by awaing a new stream.