# Agora ConvoAI Speech-to-Video Integration

This repository provides a generic protocol and implementation for external services to receive an AI agent's speech output from Agora's ConvoAI platform, enabling real-time generation of video and publishing both the audio and video back into an Agora channel for low latency, global distribution. This creates interactive experiences such as AI-powered avatars, interactive movies, and other real-time multimedia applications.

## Overview

The integration follows a three-phase workflow:

1. **Connection Setup** - Establish session and obtain WebSocket connection details
2. **Audio Streaming** - Receive real-time audio data from ConvoAI via WebSocket
3. **Video Publishing** - Generate and publish audio/video content back to Agora channel

## Architecture Flow

<img src="sequence-diagram.svg" alt="Agora ConvoAI Integration Sequence Diagram" width="100%">

## Implementation Components

### 1. Connection Setup API
[📁 connection-setup/](./connection-setup/)

### 2. WebSocket Audio Streaming
[📁 websocket-receive-audio/](./websocket-receive-audio/)

### 3. Go Audio/Video Publishing
[📁 go-publish-video/](./go-publish-video/)


## Use Cases

**Interactive AI Avatars**
- Brand mascots and talking cartoon characters for entertainment and marketing
- Virtual assistants and customer service representatives with lifelike appearance

**Dynamic Content Creation**
- Interactive movies and choose-your-own-adventure experiences that adapt to viewer input
- Automated video hosts for news, podcasts, and live streaming

**Real-time Visualization**
- Architectural design consultations with live 3D building and interior visualizations
- Scientific simulations showing molecular interactions, physics concepts, and biological processes

**Educational & Training Applications**
- AI tutors with visual demonstrations for personalized education
- Medical procedure training with anatomical models and surgical simulations

**Creative & Analytical Presentation**
- Data visualization for financial analysis, weather forecasting, and business intelligence
- Virtual real estate tours and travel experiences with immersive environments

This system transforms any descriptive AI output into corresponding visual content in real-time, making it versatile for applications where visual understanding enhances spoken explanations.
