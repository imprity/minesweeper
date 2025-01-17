declare var Go : any
declare var MINESWEEPER_WASM_SIZE : number

(async ()=>{
    async function loadData(
        path : string,
        knownDataSize : number,
        onProgress : (readData : number, dataSize : number, contentLengthMissing : boolean) => void
    ) : Promise<Uint8Array>{
        const response = await fetch(path)

        let dataSize : number = parseInt(response.headers.get('Content-Length'))
        let contentLengthMissing : boolean = false

        if (isNaN(dataSize)) {
            dataSize = knownDataSize
            contentLengthMissing = true
        }

        let data = []
        let dataRead : number = 0

        const reader = response.body.getReader()

        while (true) {
            const bodyRead = await reader.read()

            if (bodyRead.done) {
                break
            }else {
                dataRead += bodyRead.value.length

                for (let i=0; i<bodyRead.value.length; i++) {
                    data.push(bodyRead.value[i])
                }

                onProgress(dataRead, dataSize, contentLengthMissing)
            }
        }

        return new Uint8Array(data)
    }

    const loadingScreen = document.getElementById('loading-screen')

    const progressBar = document.getElementById('progress-bar')
    const progressFill = document.getElementById('progress-fill')

    const errorTextBox = document.getElementById('error-text-box')
    const errorText = document.getElementById('error-text')

    try {
        const data = await loadData(
            'minesweeper.wasm',
            MINESWEEPER_WASM_SIZE,
            (readData : number, dataSize : number, contentLengthMissing : boolean) => {
                const progress = readData / dataSize
                progressFill.style.width = `${progress * 100}%`
            }
        )

        document.body.removeChild(loadingScreen)

        const go = new Go()
        WebAssembly.instantiate(new Uint8Array(data), go.importObject).then(instnace=>{
            go.run(instnace.instance)
        })
    } catch(err) {
        errorTextBox.style.display = 'block' // show errorTextBox
        errorText.innerText = err.toString()
        console.error(err)
    }
})()
