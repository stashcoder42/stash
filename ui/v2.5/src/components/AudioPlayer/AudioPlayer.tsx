import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import videojs, { VideoJsPlayer, VideoJsPlayerOptions } from "video.js";
import useScript from "src/hooks/useScript";
import "videojs-contrib-dash";
import "videojs-mobile-ui";
import "videojs-seek-buttons";
import "./live";
import "./PlaylistButtons";
import "../ScenePlayer/persist-volume";
import MarkersPlugin, { type IMarker } from "../ScenePlayer/markers";
void MarkersPlugin;
import "../ScenePlayer/big-buttons";
import "../ScenePlayer/track-activity";
import cx from "classnames";
import {
  useAudioSaveActivity,
  useAudioIncrementPlayCount,
} from "src/core/StashService";

import * as GQL from "src/core/generated-graphql";
import { useConfigurationContext } from "src/hooks/Config";
import { VIDEO_PLAYER_ID } from "../ScenePlayer/util";

// @ts-expect-error
import airplay from "@silvermine/videojs-airplay";
// @ts-expect-error
import chromecast from "@silvermine/videojs-chromecast";
import abLoopPlugin from "videojs-abloop";
import ScreenUtils from "src/utils/screen";
import { PatchComponent } from "src/patch";
import { languageMap } from "src/utils/caption";

// register videojs plugins with error handling
try {
  airplay(videojs);
} catch (error) {
  console.warn(
    "Failed to register VideoJS Airplay plugin in AudioPlayer:",
    error
  );
}

try {
  chromecast(videojs);
} catch (error) {
  console.warn(
    "Failed to register VideoJS Chromecast plugin in AudioPlayer:",
    error
  );
}

try {
  abLoopPlugin(window, videojs);
} catch (error) {
  console.warn(
    "Failed to register VideoJS AB Loop plugin in AudioPlayer:",
    error
  );
}

function handleHotkeys(player: VideoJsPlayer, event: videojs.KeyboardEvent) {
  function seekStep(step: number) {
    const time = player.currentTime() + step;
    const duration = player.duration();
    if (time < 0) {
      player.currentTime(0);
    } else if (time < duration) {
      player.currentTime(time);
    } else {
      player.currentTime(duration);
    }
  }

  function seekPercent(percent: number) {
    const duration = player.duration();
    const time = duration * percent;
    player.currentTime(time);
  }

  function seekPercentRelative(percent: number) {
    const duration = player.duration();
    const currentTime = player.currentTime();
    const time = currentTime + duration * percent;
    if (time > duration) return;
    player.currentTime(time);
  }

  function toggleABLooping() {
    const opts = player.abLoopPlugin.getOptions();
    if (!opts.start) {
      opts.start = player.currentTime();
    } else if (!opts.end) {
      opts.end = player.currentTime();
      opts.enabled = true;
    } else {
      opts.start = 0;
      opts.end = 0;
      opts.enabled = false;
    }
    player.abLoopPlugin.setOptions(opts);
  }

  let seekFactor = 10;
  if (event.shiftKey) {
    seekFactor = 5;
  } else if (event.ctrlKey || event.altKey) {
    seekFactor = 60;
  }
  switch (event.which) {
    case 39: // right arrow
      seekStep(seekFactor);
      break;
    case 37: // left arrow
      seekStep(-seekFactor);
      break;
  }

  // toggle player looping with shift+l
  if (event.shiftKey && event.which === 76) {
    player.loop(!player.loop());
    return;
  }

  if (event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) {
    return;
  }

  switch (event.which) {
    case 32: // space
    case 13: // enter
      if (player.paused()) player.play();
      else player.pause();
      break;
    case 77: // m
      player.muted(!player.muted());
      break;
    case 70: // f
      if (player.isFullscreen()) player.exitFullscreen();
      else player.requestFullscreen();
      break;
    case 76: // l
      toggleABLooping();
      break;
    case 38: // up arrow
      player.volume(player.volume() + 0.1);
      break;
    case 40: // down arrow
      player.volume(player.volume() - 0.1);
      break;
    case 48: // 0
      player.currentTime(0);
      break;
    case 49: // 1
      seekPercent(0.1);
      break;
    case 50: // 2
      seekPercent(0.2);
      break;
    case 51: // 3
      seekPercent(0.3);
      break;
    case 52: // 4
      seekPercent(0.4);
      break;
    case 53: // 5
      seekPercent(0.5);
      break;
    case 54: // 6
      seekPercent(0.6);
      break;
    case 55: // 7
      seekPercent(0.7);
      break;
    case 56: // 8
      seekPercent(0.8);
      break;
    case 57: // 9
      seekPercent(0.9);
      break;
    case 221: // ]
      seekPercentRelative(0.1);
      break;
    case 219: // [
      seekPercentRelative(-0.1);
      break;
  }
}

type MarkerFragment = Pick<GQL.AudioMarker, "title" | "seconds"> & {
  primary_tag: Pick<GQL.Tag, "name">;
  tags: Array<Pick<GQL.Tag, "name">>;
};

function getMarkerTitle(marker: MarkerFragment) {
  if (marker.title) {
    return marker.title;
  }

  let ret = marker.primary_tag.name;
  if (marker.tags.length) {
    ret += `, ${marker.tags.map((t) => t.name).join(", ")}`;
  }

  return ret;
}

interface IAudioPlayerProps {
  audio: GQL.AudioDataFragment;
  hideScrubberOverride: boolean;
  autoplay?: boolean;
  permitLoop?: boolean;
  initialTimestamp: number;
  sendSetTimestamp: (setTimestamp: (value: number) => void) => void;
  onComplete: () => void;
  onNext: () => void;
  onPrevious: () => void;
}

export const AudioPlayer: React.FC<IAudioPlayerProps> = PatchComponent(
  "AudioPlayer",
  ({
    audio,
    hideScrubberOverride: _hideScrubberOverride,
    autoplay: _autoplay,
    permitLoop = true,
    initialTimestamp: _initialTimestamp,
    sendSetTimestamp,
    onComplete,
    onNext,
    onPrevious,
  }) => {
    const { configuration } = useConfigurationContext();
    const interfaceConfig = configuration?.interface;
    const uiConfig = configuration?.ui;
    const videoRef = useRef<HTMLDivElement>(null);
    const [_player, setPlayer] = useState<VideoJsPlayer>();
    const audioId = useRef<string>();
    const [audioSaveActivity] = useAudioSaveActivity();
    const [audioIncrementPlayCount] = useAudioIncrementPlayCount();

    const [ready, setReady] = useState(false);

    const started = useRef(false);
    const auto = useRef(false);
    const minimumPlayPercent = uiConfig?.minimumPlayPercent ?? 0;
    const trackActivity = uiConfig?.trackActivity ?? true;

    useScript(
      "https://www.gstatic.com/cv/js/sender/v1/cast_sender.js?loadCastFramework=1",
      uiConfig?.enableChromecast
    );

    const file = useMemo(
      () => (audio.files.length > 0 ? audio.files[0] : undefined),
      [audio]
    );

    const fileDuration = file?.duration;

    const maxLoopDuration = interfaceConfig?.maximumLoopDuration ?? 0;
    const looping = useMemo(
      () =>
        !!file?.duration &&
        permitLoop &&
        maxLoopDuration !== 0 &&
        file.duration < maxLoopDuration,
      [file, permitLoop, maxLoopDuration]
    );

    const getPlayer = useCallback(() => {
      if (!_player) return null;
      if (_player.isDisposed()) return null;
      return _player;
    }, [_player]);

    useEffect(() => {
      sendSetTimestamp((value: number) => {
        const player = getPlayer();
        if (player && value >= 0) {
          if (player.hasStarted() && player.paused()) {
            player.currentTime(value);
          } else {
            player.play()?.then(() => {
              player.currentTime(value);
            });
          }
        }
      });
    }, [sendSetTimestamp, getPlayer]);

    // Initialize VideoJS player
    useEffect(() => {
      const options: VideoJsPlayerOptions = {
        id: VIDEO_PLAYER_ID,
        controls: true,
        controlBar: {
          pictureInPictureToggle: false,
          volumePanel: {
            inline: false,
          },
          chaptersButton: false,
        },
        html5: {
          dash: {
            updateSettings: [
              {
                streaming: {
                  buffer: {
                    bufferTimeAtTopQuality: 30,
                    bufferTimeAtTopQualityLongForm: 30,
                  },
                  gaps: {
                    jumpGaps: false,
                    jumpLargeGaps: false,
                  },
                },
              },
            ],
          },
        },
        nativeControlsForTouch: false,
        playbackRates: [0.25, 0.5, 0.75, 1, 1.25, 1.5, 1.75, 2],
        inactivityTimeout: 2000,
        preload: "auto",
        playsinline: true,
        techOrder: ["chromecast", "html5"],
        userActions: {
          hotkeys: function (this: VideoJsPlayer, event) {
            handleHotkeys(this, event);
          },
        },
        plugins: {
          airPlay: {},
          chromecast: {},
          markers: {},
          persistVolume: {},
          bigButtons: {},
          seekButtons: {
            forward: 10,
            back: 10,
          },
          skipButtons: {},
          trackActivity: {},
          abLoopPlugin: {
            start: 0,
            end: false,
            enabled: false,
            loopIfBeforeStart: true,
            loopIfAfterEnd: true,
            pauseAfterLooping: false,
            pauseBeforeLooping: false,
            createButtons: uiConfig?.showAbLoopControls ?? false,
          },
        },
      };

      const videoEl = document.createElement("video-js");
      videoEl.setAttribute("data-vjs-player", "true");
      videoEl.setAttribute("crossorigin", "anonymous");
      videoEl.classList.add("vjs-big-play-centered");
      videoRef.current!.appendChild(videoEl);

      const vjs = videojs(videoEl, options);

      /* biome-ignore lint/suspicious/noExplicitAny: intentional */
      const settings = (vjs as any).textTrackSettings;
      settings.setValues({
        backgroundColor: "#000",
        backgroundOpacity: "0.5",
      });
      settings.updateDisplay();

      vjs.focus();
      setPlayer(vjs);

      // Video player destructor
      return () => {
        vjs.dispose();
        videoEl.remove();
        setPlayer(undefined);

        // reset audioId to force reload sources
        audioId.current = undefined;
      };
      // empty deps - only init once
      // showAbLoopControls is necessary to re-init the player when the config changes
    }, [uiConfig?.showAbLoopControls]);

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;
      const skipButtons = player.skipButtons();
      skipButtons.setForwardHandler(onNext);
      skipButtons.setBackwardHandler(onPrevious);
    }, [getPlayer, onNext, onPrevious]);

    // Player event handlers
    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      function canplay(this: VideoJsPlayer) {
        // if we're seeking before starting, don't set the initial timestamp
        // when starting from the beginning, there is a small delay before the event
        // is triggered, so we can't just check if the time is 0
        if (this.currentTime() >= 0.1) {
          return;
        }
      }

      function playing(this: VideoJsPlayer) {
        // This still runs even if autoplay failed on Safari,
        // only set flag if actually playing
        if (!started.current && !this.paused()) {
          started.current = true;
        }
      }

      function loadstart(this: VideoJsPlayer) {
        setReady(true);
      }

      player.on("canplay", canplay);
      player.on("playing", playing);
      player.on("loadstart", loadstart);

      return () => {
        player.off("canplay", canplay);
        player.off("playing", playing);
        player.off("loadstart", loadstart);
      };
    }, [getPlayer]);

    // delay before second play event after a play event to adjust for video player issues

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      // don't re-initialise the player unless the audio has changed
      if (!file || audio.id === audioId.current) return;

      audioId.current = audio.id;

      setReady(false);

      // reset on new audio
      player.trackActivity().reset();

      // DEBUG: Log the entire audio object to see what we're receiving
      // Set the audio source directly using audioStreams with correct MIME types
      if (audio.audioStreams && audio.audioStreams.length > 0) {
        // Use the first (primary) audio stream with correct MIME type
        const primaryStream = audio.audioStreams[0];

        player.src({
          src: primaryStream.url,
          type: primaryStream.mime_type || "audio/*",
        });
        player.load();

        // Set duration from file metadata if available
        if (fileDuration && fileDuration > 0) {
          // Try multiple events to ensure duration is set
          const setDuration = () => {
            try {
              player.duration(fileDuration);
              player.trigger("durationchange");
              console.log(
                `AudioPlayer: Set duration to ${fileDuration} seconds`
              );
            } catch (error) {
              console.warn("AudioPlayer: Failed to set duration:", error);
            }
          };

          player.one("loadedmetadata", setDuration);
          player.one("canplay", setDuration);
          player.one("loadstart", setDuration);
        }
      } else if (audio.paths.stream) {
        // Fallback to legacy paths if audioStreams not available
        player.src({
          src: audio.paths.stream,
          type: "audio/*",
        });
        player.load();

        // Set duration from file metadata if available
        if (fileDuration && fileDuration > 0) {
          // Try multiple events to ensure duration is set
          const setDuration = () => {
            try {
              player.duration(fileDuration);
              player.trigger("durationchange");
              console.log(
                `AudioPlayer: Set duration to ${fileDuration} seconds`
              );
            } catch (error) {
              console.warn("AudioPlayer: Failed to set duration:", error);
            }
          };

          player.one("loadedmetadata", setDuration);
          player.one("canplay", setDuration);
          player.one("loadstart", setDuration);
        }
      }
    }, [audio, file, getPlayer, fileDuration]);

    // Load captions when audio changes
    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      // Remove existing text tracks
      const remoteTracks = player.remoteTextTracks();
      while (remoteTracks && remoteTracks.length > 0) {
        player.removeRemoteTextTrack(
          remoteTracks[0] as unknown as HTMLTrackElement
        );
      }

      if (
        !audio.captions ||
        audio.captions.length === 0 ||
        !audio.paths.caption
      )
        return;

      function getDefaultLanguageCode() {
        let languageCode = window.navigator.language;

        if (languageCode.indexOf("-") !== -1) {
          languageCode = languageCode.split("-")[0];
        }

        if (languageCode.indexOf("_") !== -1) {
          languageCode = languageCode.split("_")[0];
        }

        return languageCode;
      }

      const defaultLang = getDefaultLanguageCode();
      let hasDefault = false;

      for (const caption of audio.captions) {
        const lang = caption.language_code;
        let label = lang;
        if (languageMap.has(lang)) {
          label = languageMap.get(lang)!;
        }

        label = label + " (" + caption.caption_type + ")";
        const setAsDefault = !hasDefault && defaultLang === lang;
        if (setAsDefault) {
          hasDefault = true;
        }

        player.addRemoteTextTrack(
          {
            kind: "captions",
            language: lang,
            label: label,
            src: `${audio.paths.caption}?lang=${lang}&type=${caption.caption_type}`,
            default: setAsDefault,
          },
          false
        );
      }
    }, [audio, getPlayer]);

    const loadMarkers = useCallback(() => {
      const player = getPlayer();
      if (!player) return;
      if (!audio.audio_markers) return;

      const markerData = audio.audio_markers.map((marker) => ({
        title: getMarkerTitle(marker),
        seconds: marker.seconds,
        end_seconds: marker.end_seconds ?? null,
        primaryTag: marker.primary_tag,
      }));

      const markers = player!.markers();

      const uniqueTagNames = markerData
        .map((marker) => marker.primaryTag.name)
        .filter((value, index, self) => self.indexOf(value) === index);

      // Wait for colors
      markers.findColors(uniqueTagNames);

      const showRangeTags =
        !ScreenUtils.isMobile() && (uiConfig?.showRangeMarkers ?? true);
      const timestampMarkers: IMarker[] = [];
      const rangeMarkers: IMarker[] = [];

      if (!showRangeTags) {
        for (const marker of markerData) {
          timestampMarkers.push(marker);
        }
      } else {
        for (const marker of markerData) {
          if (marker.end_seconds === null) {
            timestampMarkers.push(marker);
          } else {
            rangeMarkers.push(marker);
          }
        }
      }

      requestAnimationFrame(() => {
        markers.addDotMarkers(timestampMarkers);
        markers.addRangeMarkers(rangeMarkers);
      });
    }, [getPlayer, audio, uiConfig]);

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      if (audio.paths.cover) {
        player.poster(audio.paths.cover);
      } else {
        player.poster("");
      }

      // Define the event handler outside the useEffect
      const handleLoadMetadata = () => {
        loadMarkers();
      };

      // Ensure markers are added after player is fully ready and sources are loaded
      if (player.readyState() >= 1) {
        loadMarkers();
      } else {
        player.on("loadedmetadata", handleLoadMetadata);
      }

      return () => {
        player.off("loadedmetadata", handleLoadMetadata);
        const markers = player!.markers();
        markers.clearMarkers();
      };
    }, [getPlayer, audio, loadMarkers]);

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      async function saveActivity(resumeTime: number, playDuration: number) {
        if (!audio.id) return;

        await audioSaveActivity({
          variables: {
            id: audio.id,
            playDuration,
            resume_time: resumeTime,
          },
        });
      }

      async function incrementPlayCount() {
        if (!audio.id) return;

        await audioIncrementPlayCount({
          variables: {
            id: audio.id,
          },
        });
      }

      const activity = player.trackActivity();
      activity.saveActivity = saveActivity;
      activity.incrementPlayCount = incrementPlayCount;
      activity.minimumPlayPercent = minimumPlayPercent;
      activity.setEnabled(trackActivity);
    }, [
      getPlayer,
      audio,
      trackActivity,
      minimumPlayPercent,
      audioIncrementPlayCount,
      audioSaveActivity,
    ]);

    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      player.loop(looping);
    }, [getPlayer, looping]);

    // biome-ignore lint/correctness/useExhaustiveDependencies: audio is the trigger for re-running autoplay when the track changes
    useEffect(() => {
      const player = getPlayer();
      if (!player || !ready || !auto.current) {
        return;
      }

      player.play();
      auto.current = false;
    }, [getPlayer, audio, ready]);

    // Attach handler for onComplete event
    useEffect(() => {
      const player = getPlayer();
      if (!player) return;

      player.on("ended", onComplete);

      return () => player.off("ended");
    }, [getPlayer, onComplete]);

    return (
      <div
        className={cx("VideoPlayer", {
          "no-file": !file,
        })}
      >
        <div className="video-wrapper" ref={videoRef} />
      </div>
    );
  }
);

export default AudioPlayer;
