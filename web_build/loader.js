var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
(() => __awaiter(this, void 0, void 0, function* () {
    function loadData(path, knownDataSize, onProgress) {
        return __awaiter(this, void 0, void 0, function* () {
            const response = yield fetch(path);
            let dataSize = parseInt(response.headers.get('Content-Length'));
            let contentLengthMissing = false;
            if (isNaN(dataSize)) {
                dataSize = knownDataSize;
                contentLengthMissing = true;
            }
            let data = [];
            let dataRead = 0;
            const reader = response.body.getReader();
            while (true) {
                const bodyRead = yield reader.read();
                if (bodyRead.done) {
                    break;
                }
                else {
                    dataRead += bodyRead.value.length;
                    for (let i = 0; i < bodyRead.value.length; i++) {
                        data.push(bodyRead.value[i]);
                    }
                    onProgress(dataRead, dataSize, contentLengthMissing);
                }
            }
            return new Uint8Array(data);
        });
    }
    // get document elements
    const loadingScreen = document.getElementById('loading-screen');
    const progressBar = document.getElementById('progress-bar');
    const progressFill = document.getElementById('progress-fill');
    const loadingErrorContainer = document.getElementById('loading-error-container');
    const loadingErrorText = document.getElementById('loading-error-text');
    const crashScreen = document.getElementById('crash-screen');
    const crashErrortext = document.getElementById('crash-error-text');
    // load wasm
    try {
        const data = yield loadData('minesweeper.wasm', MINESWEEPER_WASM_SIZE, (readData, dataSize, contentLengthMissing) => {
            const progress = readData / dataSize;
            progressFill.style.width = `${progress * 100}%`;
        });
        document.body.removeChild(loadingScreen);
        // before instantiating go wasm
        // override few methods and syscalls
        function countNewLine(str) {
            let counter = 0;
            let index = 0;
            while (true) {
                const newIndex = str.indexOf("\n", index);
                if (newIndex < 0) {
                    break;
                }
                else {
                    counter++;
                }
                index = newIndex + 1;
            }
            return counter;
        }
        const originalWriteSync = globalThis.fs.writeSync;
        const decoder = new TextDecoder("utf-8");
        let logText = "";
        const logTextMax = 500;
        let newLineCounter = 0;
        globalThis.fs.writeSync = function (fd, buf) {
            let bufText = decoder.decode(buf);
            const newLineCount = countNewLine(bufText);
            newLineCounter += newLineCount;
            logText += bufText;
            if (newLineCounter > logTextMax) {
                const diff = newLineCounter - logTextMax;
                let index = 0;
                for (let i = 0; i < diff; i++) {
                    const newIndex = logText.indexOf("\n", index);
                    if (newIndex < 0) {
                        break;
                    }
                    index = newIndex + 1;
                }
                if (index < 0) {
                    index = 0;
                }
                logText = logText.substring(index);
                newLineCounter = logTextMax;
            }
            return originalWriteSync(fd, buf);
        };
        const go = new Go();
        go.exit = (code) => {
            if (code != 0) {
                crashScreen.style.display = 'flex'; // show errorTextBox
                crashErrortext.innerText = logText;
            }
        };
        WebAssembly.instantiate(new Uint8Array(data), go.importObject).then(instnace => {
            go.run(instnace.instance);
        });
    }
    catch (err) {
        loadingErrorContainer.style.display = 'block'; // show errorTextBox
        loadingErrorText.innerText = err.toString();
        console.error(err);
    }
}))();
