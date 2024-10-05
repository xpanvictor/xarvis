# Project Xarvis
**Xpanvictor**

Xarvis is an AI-powered virtual assistant inspired by Iron Man's Jarvis, designed to execute commands, control devices, manage tasks, and integrate with various systems. It supports speech-to-text, text-to-speech, and NLP-based command execution, with a modular architecture for flexibility and scalability.

> Astounded, life carries me, left with no remorse, I can only hope to breath while I await death.
Halo, Xarvis to save as much as possible, time flees.


## Table of Contents
1. [Project Overview](#project-overview)
2. [System Architecture](#system-architecture)
3. [Features](#features)
4. [Backbone](#backbone)
5. [Core Components](#core-components)
6. [Setup](#setup)
7. [Usage](#usage)
8. [Contributing](#contributing)
9. [License](#license)

## 1. Project Overview

Xarvis is a modular AI system designed to automate tasks, control devices, and assist in daily activities. It integrates natural language processing (NLP), speech recognition, and text-to-speech functionalities, while also serving as a platform for automation, device control, and task management. Xarvis's architecture is designed to be flexible, scalable, and adaptable for future functionalities.

## 2. System Architecture

Xarvis is composed of two key layers:

- **Backbone:**
    1. General Proxy and Pipeline Server
    2. Job Runner
    3. Aggregator
    4. Brain (Intensive Calculation and Code Generation)

- **Core Components:**
    1. Natural Language Processing (NLP)
    2. Speech Recognition
    3. Text-to-Speech (TTS)
    4. Internal Server System (ISS)
    5. Command Execution Layer (CEL)
    6. Integration Layer (IL)
        - Visual Layer
    7. Automations

## 3. Features
- **Voice Interaction:** Speech recognition and text-to-speech modules for interacting with the system via voice commands.
- **NLP Command Parsing:** NLP engine for understanding commands and executing corresponding actions.
- **Device Control:** Integration with ESP32-CAM and other devices for hardware interaction.
- **Automation:** Define custom automations for task scheduling and reminders.
- **Modular Design:** Easy to add new features and expand functionality.
- **Custom Command Execution:** Ability to run system commands and scripts based on voice input.

## 4. Backbone

- **General Proxy and Pipeline Server:** A server that handles communication between different modules and the external world.
- **Job Runner:** Manages tasks and processes commands in a queue.
- **Aggregator:** Collects information from various sources, integrates data, and passes it to the Brain.
- **Brain (Intensive Calculation and Code Generation):** The core computational unit responsible for decision-making, complex calculations, and generating code where necessary.

## 5. Core Components

- **Natural Language Processing (NLP):** This module processes user commands and converts them into actionable tasks.
- **Speech Recognition:** Converts voice input into text using STT (Speech-to-Text) technology.
- **Text-to-Speech (TTS):** Converts system responses or generated text into spoken output.
- **Internal Server System (ISS):** Manages server requests and communication between modules.
- **Command Execution Layer (CEL):** Executes system commands or predefined tasks based on parsed NLP results.
- **Integration Layer (IL):** Includes both a visual layer for feedback and automation handling for system integration.
- **Automations:** Users can schedule tasks, set reminders, and automate specific activities.

## 6. Setup

### Requirements:
- Python 3.x
- ESP32-CAM module
- Microphone (for voice input)
- Bluetooth devices (for TTS playback)

### Install Dependencies:

Clone the repository:

```bash
git clone https://github.com/your-repo/xarvis.git
cd xarvis
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

## 7. Usage
```bash
python main.py
```
Could also use make.

## 8. Contributing
Feel free to send PRs.

## 9. Licence
Refer to 

