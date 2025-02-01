class InternalPlayer {
    constructor(buffer) {
        this._id = 0;
        this._buffer = null;
        this._source = null;
        this._gain = null;
        this._playHead = 0;
        this._playStartedAt = 0;
        this._state = 'Paused';
        this._endHandler = null;
        InternalPlayer._idCounter += 1;
        this._id = InternalPlayer._idCounter;
        this._buffer = buffer;
        this._gain = InternalPlayer.audioContext.createGain();
        this._gain.connect(InternalPlayer.audioContext.destination);
        this._endHandler = () => {
            if (this._state === 'Playing') {
                this._state = 'Paused';
                this._playHead = this._buffer.duration;
            }
            this._source = null;
        };
    }
    id() {
        return this._id;
    }
    state() {
        return this._state;
    }
    play() {
        if (this._state === 'Playing') {
            return;
        }
        this._state = 'Playing';
        this._source = InternalPlayer.audioContext.createBufferSource();
        this._source.buffer = this._buffer;
        this._source.connect(this._gain);
        this._source.addEventListener('ended', this._endHandler);
        this._playStartedAt = InternalPlayer.audioContext.currentTime;
        this._source.start(this._playStartedAt, this._playHead);
    }
    pause() {
        if (this._state === 'Paused') {
            return;
        }
        this._state = 'Paused';
        this._playHead += InternalPlayer.audioContext.currentTime - this._playStartedAt;
        this._source.stop();
        this._source = null;
    }
    duration() {
        return this._buffer.duration;
    }
    position() {
        if (this._state === 'Playing') {
            return InternalPlayer.audioContext.currentTime - this._playStartedAt + this._playHead;
        }
        return this._playHead;
    }
    setPosition(at) {
        if (at < 0) {
            at = 0;
        }
        if (at > this.duration()) {
            at = this.duration();
        }
        if (this._state === 'Playing') {
            console.log('removeEventListener');
            this._source.removeEventListener('ended', this._endHandler);
            this._source.stop();
            this._source = null;
            this._source = InternalPlayer.audioContext.createBufferSource();
            this._source.buffer = this._buffer;
            this._source.connect(this._gain);
            this._source.addEventListener('ended', this._endHandler);
            this._playHead = at;
            this._playStartedAt = InternalPlayer.audioContext.currentTime;
            this._source.start(this._playStartedAt, this._playHead);
        }
        else {
            this._playHead = at;
        }
    }
    volume() {
        return this._gain.gain.value;
    }
    setVolume(volume) {
        if (volume < 0) {
            volume = 0;
        }
        if (volume > 1) {
            volume = 1;
        }
        this._gain.gain.setValueAtTime(volume, InternalPlayer.audioContext.currentTime);
    }
}
InternalPlayer._idCounter = 0;
InternalPlayer.audioContext = null;
let AUDIO_BUFFERS = {};
let INTERNAL_PLAYERS = [];
let AUDIO_CONTEXT = null;
function initAudioContext(sampleRate) {
    let audioContext = new (window['AudioContext'] || window['webkitAudioContext'])({ sampleRate: sampleRate });
    AUDIO_CONTEXT = audioContext;
    InternalPlayer.audioContext = audioContext;
    const events = ["touchend", "keyup", "mouseup"];
    let callback;
    let calledOnAudioResume = false;
    const removeCallbacks = () => {
        events.forEach((toRemove) => {
            document.removeEventListener(toRemove, callback);
        });
    };
    callback = () => {
        AUDIO_CONTEXT.resume().then(() => {
            if (typeof ON_AUDIO_RESUME === 'function') {
                if (!calledOnAudioResume) {
                    calledOnAudioResume = true;
                    ON_AUDIO_RESUME();
                }
                if (calledOnAudioResume) {
                    removeCallbacks();
                }
            }
        });
    };
    events.forEach(toAdd => {
        document.addEventListener(toAdd, callback);
    });
}
// =================================
// buffer functions
// =================================
function newBufferFromAudioFile(name, file, onDecoded) {
    try {
        const promise = AUDIO_CONTEXT.decodeAudioData(file);
        promise.then((decoded) => {
            AUDIO_BUFFERS[name] = decoded;
            onDecoded(true);
        });
        promise.catch(err => {
            console.error(`failed to decode ${name} : ${err}`);
            onDecoded(false);
        });
    }
    catch (err) {
        console.error(`failed to decode ${name} : ${err}`);
        onDecoded(false);
    }
}
function newBufferFromUndecodedAudioFile(name, channelDatas, sampleRate) {
    let channelByteLength = 0;
    if (channelDatas.length > 0) {
        channelByteLength = channelDatas[0].byteLength;
    }
    const buffer = AUDIO_CONTEXT.createBuffer(channelDatas.length, channelByteLength, sampleRate);
    for (let i = 0; i < channelDatas.length; i++) {
        buffer.copyToChannel(channelDatas[i], i);
    }
    AUDIO_BUFFERS[name] = buffer;
}
// =================================
// player functions
// =================================
function newPlayer(buffer) {
    const p = new InternalPlayer(AUDIO_BUFFERS[buffer]);
    INTERNAL_PLAYERS[p.id()] = p;
    return p.id();
}
function playerIsPlaying(playerId) {
    return INTERNAL_PLAYERS[playerId].state() == 'Playing';
}
function playerPlay(playerId) {
    INTERNAL_PLAYERS[playerId].play();
}
function playerPause(playerId) {
    INTERNAL_PLAYERS[playerId].pause();
}
function playerDuration(playerId) {
    return INTERNAL_PLAYERS[playerId].duration();
}
function playerPosition(playerId) {
    return INTERNAL_PLAYERS[playerId].position();
}
function playerSetPosition(playerId, at) {
    return INTERNAL_PLAYERS[playerId].setPosition(at);
}
function playerVolume(playerId) {
    return INTERNAL_PLAYERS[playerId].volume();
}
function playerSetVolume(playerId, volume) {
    return INTERNAL_PLAYERS[playerId].setVolume(volume);
}
