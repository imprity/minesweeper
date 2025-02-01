interface BufferMap {
    [key : string] : AudioBuffer
}

interface InternalPlayerMap {
    [key : number] : InternalPlayer
}

let AUDIO_BUFFERS : BufferMap = {}
let INTERNAL_PLAYERS : InternalPlayerMap = []
let AUDIO_CONTEXT : AudioContext = null

declare var ON_AUDIO_RESUME : ()=>void

function initAudioContext(sampleRate : number) {
    let audioContext = new (window['AudioContext'] || window['webkitAudioContext'])({sampleRate : sampleRate})

    AUDIO_CONTEXT = audioContext
    InternalPlayer.audioContext = audioContext

    const events = ["touchend", "keyup", "mouseup"]

    let callback : ()=>void
    let calledOnAudioResume : boolean = false

    const removeCallbacks = () => {
        events.forEach((toRemove)=>{
            document.removeEventListener(toRemove, callback)
        })
    }

    callback = () => {
        AUDIO_CONTEXT.resume().then(()=>{
            if (typeof ON_AUDIO_RESUME === 'function') {
                if (!calledOnAudioResume) {
                    calledOnAudioResume = true
                    ON_AUDIO_RESUME()
                }

                if (calledOnAudioResume) {
                    removeCallbacks()
                }
            }
        })
    }

    events.forEach(toAdd=>{
        document.addEventListener(toAdd, callback)
    })
}

// =================================
// buffer functions
// =================================
function newBufferFromAudioFile(
    name : string,
    file : ArrayBuffer,
    onDecoded : (success : boolean)=>void,
){
    try {
        const promise = AUDIO_CONTEXT.decodeAudioData(file)
        promise.then((decoded : AudioBuffer) => {
            AUDIO_BUFFERS[name] = decoded
            onDecoded(true)
        })
        promise.catch(err=>{
            console.error(`failed to decode ${name} : ${err}`)
            onDecoded(false)
        })
    }catch (err) {
        console.error(`failed to decode ${name} : ${err}`)
        onDecoded(false)
    }
}

function newBufferFromUndecodedAudioFile(
    name : string,
    channelDatas : Array<Float32Array>,
    sampleRate : number,
) {
    let channelByteLength = 0

    if (channelDatas.length > 0) {
        channelByteLength = channelDatas[0].byteLength
    }

    const buffer = AUDIO_CONTEXT.createBuffer(
        channelDatas.length, channelByteLength, sampleRate);

    for (let i=0; i<channelDatas.length; i++) {
        buffer.copyToChannel(channelDatas[i], i)
    }

    AUDIO_BUFFERS[name] = buffer
}

// =================================
// player functions
// =================================
function newPlayer(buffer : string) : number{
    const p = new InternalPlayer(AUDIO_BUFFERS[buffer])

    INTERNAL_PLAYERS[p.id()] = p

    return p.id()
}

function playerIsPlaying(playerId : number) : boolean {
    return INTERNAL_PLAYERS[playerId].state() == 'Playing'
}

function playerPlay(playerId : number) {
    INTERNAL_PLAYERS[playerId].play()
}

function playerPause(playerId : number) {
    INTERNAL_PLAYERS[playerId].pause()
}

function playerDuration(playerId : number) : number{
    return INTERNAL_PLAYERS[playerId].duration()
}

function playerPosition(playerId : number) : number{
    return INTERNAL_PLAYERS[playerId].position()
}

function playerSetPosition(playerId : number, at : number) {
    return INTERNAL_PLAYERS[playerId].setPosition(at)
}

function playerVolume(playerId : number) : number{
    return INTERNAL_PLAYERS[playerId].volume()
}

function playerSetVolume(playerId : number, volume : number) {
    return INTERNAL_PLAYERS[playerId].setVolume(volume)
}
