var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
//const audioContext : AudioContext = new (window['AudioContext'] || window['webkitAudioContext'])()
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
/*
function newPlayer(buffer : AudioBuffer) : Player{
    const p = new Player()

    PLAYER_ID_COUNTER += 1
    p._id = PLAYER_ID_COUNTER

    let addedPlayer :boolean = false
    for (let i=0; i<PLAYERS.length; i++) {
        if (PLAYERS[i] === null) {
            PLAYERS[i] = p
            addedPlayer = true
            break
        }
    }
    if (!addedPlayer) {
        PLAYERS.push(p)
    }

    p._buffer = buffer

    p._gain = audioContext.createGain()
    p._gain.connect(audioContext.destination)

    p._endHandler = function() {
        if (p._state == 'Playing') {
            p._state = 'Stoped'
            p._playHead = p._buffer.duration
        }

        p._source = null
    }

    return p
}
*/
function getData(path) {
    return __awaiter(this, void 0, void 0, function* () {
        const res = yield fetch(path);
        const blob = yield res.blob();
        return blob.arrayBuffer();
    });
}
/*
// TEST TEST TEST TEST TEST TEST

(async ()=> {
    const data = await getData('wobble2.wav')
    const decoded = await audioContext.decodeAudioData(data)

    const player = new InternalPlayer(decoded)

    const controlDiv = document.createElement('div')
    document.body.appendChild(controlDiv)

    const playButton = document.createElement('button')
    playButton.innerText = 'play'
    controlDiv.appendChild(playButton)

    const stopButton = document.createElement('button')
    stopButton.innerText = 'stop'
    controlDiv.appendChild(stopButton)

    const pauseButton = document.createElement('button')
    pauseButton.innerText = 'pause'
    controlDiv.appendChild(pauseButton)

    const volumeSlider = document.createElement('input')
    volumeSlider.setAttribute('type', 'range')
    volumeSlider.setAttribute('min', '0')
    volumeSlider.setAttribute('max', '1')
    volumeSlider.setAttribute('step', '0.02')
    volumeSlider.oninput = function() {
        player.setVolume(Number(volumeSlider.value))
    }
    controlDiv.appendChild(volumeSlider)

    playButton.onclick = ()=>{
        audioContext.resume()
        player.play()
    }

    stopButton.onclick = ()=>{
        player.stop()
    }

    pauseButton.onclick = ()=> {
        player.pause()
    }

    const timeSliderDiv = document.createElement('div')
    document.body.appendChild(timeSliderDiv)

    const timeSlider = document.createElement('input')
    timeSlider.setAttribute('type', 'range')
    timeSlider.setAttribute('min', '0')
    timeSlider.setAttribute('max', '1')
    timeSlider.setAttribute('step', '0.02')
    timeSliderDiv.appendChild(timeSlider)

    const timeSliderLabel = document.createElement('label')
    timeSliderLabel.innerText = '0/0'
    timeSliderDiv.appendChild(timeSliderLabel)

    const stateDisplay = document.createElement('p')
    document.body.appendChild(stateDisplay)

    let dragging : boolean = false
    let playAfterDragging : boolean = false

    timeSlider.oninput = ()=>{
        if (!dragging) { // wasn't dragging before
            playAfterDragging = player.state() == 'Playing'
        }
        dragging = true
        player.pause()
    }
    timeSlider.onchange = ()=>{
        dragging = false
        if (playAfterDragging) {
            player.play()
        }
    }

    const onRequest = ()=>{
        if (!dragging) {
            const t = player.position() / player.duration()
            timeSlider.value = String(t)
        }else {
            let t = Number(timeSlider.value)
            t = t * player.duration()
            player.setPosition(t)
        }

        timeSliderLabel.innerText = `${player.position().toFixed(3)}/${player.duration().toFixed(3)}`
        stateDisplay.innerText = player.state()

        requestAnimationFrame(onRequest)
    }

    requestAnimationFrame(onRequest)
})()

// TEST TEST TEST TEST TEST TEST
*/
let AUDIO_BUFFERS = {};
let INTERNAL_PLAYERS = [];
let AUDIO_CONTEXT = null;
function initAudioContext(sampleRate) {
    let audioContext = new (window['AudioContext'] || window['webkitAudioContext'])({ sampleRate: sampleRate });
    AUDIO_CONTEXT = audioContext;
    InternalPlayer.audioContext = audioContext;
}
(() => {
    const events = ["touchend", "keyup", "mouseup"];
    let callback;
    const removeCallbacks = () => {
        events.forEach((toRemove) => {
            document.removeEventListener(toRemove, callback);
        });
    };
    callback = () => {
        AUDIO_CONTEXT.resume().then(() => {
            if (typeof ON_AUDIO_RESUME === 'function') {
                // TEST TEST TEST TEST
                console.log('from js : ON_AUDIO_RESUME');
                // TEST TEST TEST TEST
                ON_AUDIO_RESUME();
            }
            removeCallbacks();
        });
    };
    events.forEach(toAdd => {
        document.addEventListener(toAdd, callback);
    });
})();
// =================================
// buffer functions
// =================================
function newBufferFromAudioFile(name, file, onDecoded) {
    AUDIO_CONTEXT.decodeAudioData(file, (decoded) => {
        AUDIO_BUFFERS[name] = decoded;
        onDecoded(true);
    }, () => {
        onDecoded(false);
    });
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
