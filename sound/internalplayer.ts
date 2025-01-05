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

