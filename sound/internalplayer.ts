//const audioContext : AudioContext = new (window['AudioContext'] || window['webkitAudioContext'])()

type PlayerState = 'Paused' | 'Playing'

class InternalPlayer {
    static _idCounter : number = 0
    static audioContext : AudioContext = null

    _id : number = 0

    _buffer : AudioBuffer = null

    _source : AudioBufferSourceNode = null
    _gain : GainNode = null

    _playHead : number = 0
    _playStartedAt : number = 0

    _state : PlayerState = 'Paused'

    _endHandler : () => void = null

    constructor(buffer : AudioBuffer) {
        InternalPlayer._idCounter += 1
        this._id = InternalPlayer._idCounter

        this._buffer = buffer

        this._gain = InternalPlayer.audioContext.createGain()
        this._gain.connect(InternalPlayer.audioContext.destination)

        this._endHandler = ()=> {
            if (this._state === 'Playing') {
                this._state = 'Paused'
                this._playHead = this._buffer.duration
            }

            this._source = null
        }
    }

    id() : number {
        return this._id
    }

    state() :PlayerState {
        return this._state
    }

    play () {
        if (this._state === 'Playing') {
            return
        }

        this._state = 'Playing'

        this._source = InternalPlayer.audioContext.createBufferSource()
        this._source.buffer = this._buffer
        this._source.connect(this._gain)

        this._source.addEventListener('ended', this._endHandler)

        this._playStartedAt = InternalPlayer.audioContext.currentTime
        this._source.start(this._playStartedAt, this._playHead)
    }

    pause() {
        if (this._state === 'Paused') {
            return
        }
        this._state = 'Paused'
        this._playHead += InternalPlayer.audioContext.currentTime - this._playStartedAt
        this._source.stop()
        this._source = null
    }

    duration() :number{
        return this._buffer.duration
    }

    position() :number {
        if (this._state === 'Playing') {
            return InternalPlayer.audioContext.currentTime - this._playStartedAt + this._playHead
        }
        return this._playHead
    }

    setPosition(at : number) {
        if (at < 0) {
            at = 0
        }
        if (at > this.duration()) {
            at = this.duration()
        }

        if (this._state === 'Playing') {
            console.log('removeEventListener')
            this._source.removeEventListener('ended', this._endHandler)
            this._source.stop()
            this._source = null

            this._source = InternalPlayer.audioContext.createBufferSource()
            this._source.buffer = this._buffer
            this._source.connect(this._gain)

            this._source.addEventListener('ended', this._endHandler)

            this._playHead = at
            this._playStartedAt = InternalPlayer.audioContext.currentTime
            this._source.start(this._playStartedAt, this._playHead)
        }else {
            this._playHead = at
        }
    }

    volume() : number {
        return this._gain.gain.value
    }

    setVolume(volume : number) {
        if (volume < 0) {
            volume = 0
        }
        if (volume > 1) {
            volume = 1
        }
        this._gain.gain.setValueAtTime(volume, InternalPlayer.audioContext.currentTime)
    }
}

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

async function getData(path : string) : Promise<ArrayBuffer>{
    const res = await fetch(path)
    const blob = await res.blob()
    return blob.arrayBuffer()
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
