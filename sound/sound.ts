interface BufferMap {
    [key : string] : AudioBuffer
}

interface InternalPlayerMap {
    [key : number] : InternalPlayer
}

let AUDIO_BUFFERS : BufferMap = {}
let INTERNAL_PLAYERS : InternalPlayerMap = []
let AUDIO_CONTEXT : AudioContext = null

function initAudioContext(sampleRate : number) {
    let audioContext = new (window['AudioContext'] || window['webkitAudioContext'])({sampleRate : sampleRate})

    AUDIO_CONTEXT = audioContext
    InternalPlayer.audioContext = audioContext
}

declare var ON_AUDIO_RESUME : ()=>void

(()=>{
    const events = ["touchend", "keyup", "mouseup"]

    let callback : ()=>void

    const removeCallbacks = () => {
        events.forEach((toRemove)=>{
            document.removeEventListener(toRemove, callback)
        })
    }

    callback = () => {
        AUDIO_CONTEXT.resume().then(()=>{
            if (typeof ON_AUDIO_RESUME === 'function') {
                ON_AUDIO_RESUME()
            }
            removeCallbacks()
        })
    }

    events.forEach(toAdd=>{
        document.addEventListener(toAdd, callback)
    })
})()

// =================================
// buffer functions
// =================================
function newBufferFromAudioFile(
    name : string,
    file : ArrayBuffer,
    onDecoded : (success : boolean)=>void,
){
    AUDIO_CONTEXT.decodeAudioData(
        file,
        (decoded : AudioBuffer)=>{
            AUDIO_BUFFERS[name] = decoded
            onDecoded(true)
        },
        ()=>{
            onDecoded(false)
        }
    )
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
