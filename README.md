# Voidcut

> A smart video editor that runs in the background while you record. Logs mic and keyboard
> activity in real time, then uses that activity log to cut dead air after you upload the
> recording. Built as a full event-driven distributed microservices system in Go.

## The Problem

1. **You record a coding tutorial**

   Screen recording running, you're explaining code, referring to docs on another monitor,
   looking back and forth.

2. **Dead air accumulates**

   Every time you look away, pause to think, or wait for something to load, silence and
   stillness builds up in the recording.

3. **Manual editing is painful**

   You spend hours scrubbing through footage cutting gaps. The ratio of editing time to
   recording time is brutal.

## The Solution

1. **Voidcut runs in the background**

   Before you hit record, start Voidcut. It runs silently, with no UI in the way.

2. **Real-time activity logging**

   Every mic event and keyboard/mouse event is timestamped and streamed to the
   ActivityLogger via NATS.

3. **Upload your raw video**

   When recording ends, upload the video file. Voidcut already has your activity log ready.

4. **Process and download**

   After upload, AnalysisService computes keep/cut intervals and EditingService runs FFmpeg.
   Download the edited video when processing is complete.
