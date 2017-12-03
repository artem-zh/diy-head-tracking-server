# DIY Head Tracking Server

## Overview

A part of my experimental gyro-based head tracking system (another part is
X-Plane plugin located [here][xplane-plugin-part]).

The main idea is, why not try using smartphone's gyroscope sensors for
X-Plane head tracking, because most of us have smartphone nowadays.  

In its very early state, there're still tons of things to do.

## High Level Design

The server consists of these parts now:

1. Static HTTP server, which:
   1. provides primitive web app for a smartphone;
   2. listens to WebSocket connections from a smartphone.
2. 'Free-form' TCP server. The [plugin][xplane-plugin-part] connects to this
   server in order to receive head tracking data from it. 

## Goals

* use smartphone's capabilies for head tracking;
* play with DeviceOrientation API;
* learning Go :)

## Known Issues and Limitations

1. Seems to be challenging to have a 'starting point' (to know what is 
  'sim pilot's watching straight towards monitor' point). Working on that. 
2. Currently, too much 'noise' in data. Hence, may be jaggy. I'll add smoothing
  filter to mitigate it.
3. Requires LAN (smartphone is connecting to a local 'server')
4. Hence, laggy

## Useful links

1. [xplane-plugin-part] The X-Plane plugin, 2nd part of the system.
2. [Usage of Device Orientation API] Nice dev.opera's article on the topic.
3. [DeviceOrientation Event Specification]

[xplane-plugin-part]: https://github.com/artem-zh/diy-head-tracking-xplane
[Usage of Device Orientation API]: https://dev.opera.com/articles/w3c-device-orientation-usage/
[DeviceOrientation Event Specification]: https://w3c.github.io/deviceorientation/spec-source-orientation.html
