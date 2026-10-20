/*
 * Smoe
 * @author Smoe
 * @url https://smoe.cc
 */

/* ===== Lenis 惯性平滑滚动（Lusion 式顺滑） ===== */
const smoeSmooth = () => {
    if (typeof Lenis === 'undefined') return null;
    if (matchMedia('(prefers-reduced-motion: reduce)').matches) return null;
    const lenis = new Lenis({ lerp: 0.09, smoothWheel: true });
    const raf = (t) => { lenis.raf(t); requestAnimationFrame(raf); };
    requestAnimationFrame(raf);
    return lenis;
};

/* ===== 点赞粒子迸发 ===== */
const smoeBurst = (el) => {
    const r = el.getBoundingClientRect();
    const cx = r.left + r.width / 2, cy = r.top + r.height / 2;
    for (let i = 0; i < 16; i++) {
        const s = document.createElement('i');
        s.className = 'spark';
        s.style.left = cx + 'px';
        s.style.top = cy + 'px';
        document.body.append(s);
        const a = (Math.PI * 2 * i) / 16 + Math.random() * 0.5;
        const d = 36 + Math.random() * 56;
        s.animate([
            { transform: 'translate(-50%,-50%) scale(1)', opacity: 1 },
            { transform: `translate(calc(-50% + ${(Math.cos(a) * d).toFixed(1)}px), calc(-50% + ${(Math.sin(a) * d).toFixed(1)}px)) scale(0)`, opacity: 0.4 },
        ], { duration: 500 + Math.random() * 300, easing: 'cubic-bezier(.16,1,.3,1)' }).onfinish = () => s.remove();
    }
};

/* ===== Lusion-style list stage: one WebGL context, many card meshes ===== */
const smoeStage = (() => {
    const off = { observe() {} };
    if (typeof WebGLRenderingContext === 'undefined') return off;
    if (matchMedia('(prefers-reduced-motion: reduce)').matches) return off;

    const MAX_TEXTURE_EDGE = 1400;
    const VERT = 'attribute vec2 p;uniform vec4 uRect;varying vec2 vUv;void main(){vUv=p*0.5+0.5;gl_Position=vec4(mix(uRect.x,uRect.z,vUv.x),mix(uRect.y,uRect.w,vUv.y),0.,1.);}';
    const FRAG = 'precision highp float;uniform sampler2D uColor;uniform vec2 uMouse,uSpan,uSize;uniform float uReveal,uHover,uSide,uScroll,uTime,uRadius,uAlpha;varying vec2 vUv;'
        + 'float sat(float x){return clamp(x,0.0,1.0);}float linstep(float a,float b,float x){return sat((x-a)/(b-a));}'
        + 'float hash(vec2 p){return fract(sin(dot(p,vec2(127.1,311.7)))*43758.5453123);}'
        + 'vec3 sampleColor(vec2 uv,float blur,vec2 noise){if(blur<0.0001)return texture2D(uColor,uv).rgb;vec3 c=vec3(0.0);float a=6.2831853*noise.x;'
        + 'for(int i=0;i<6;i++){float fi=float(i)+0.5;float r=sqrt(fi/6.0)*blur;a+=2.3999632;c+=texture2D(uColor,uv+vec2(cos(a),sin(a))*r).rgb;}return c/6.0;}'
        + 'float roundedAlpha(vec2 uv){vec2 px=uv*uSize;vec2 h=uSize*0.5;vec2 q=abs(px-h)-(h-vec2(uRadius));float d=length(max(q,0.0))+min(max(q.x,q.y),0.0)-uRadius;return 1.0-smoothstep(0.0,1.4,d);}'
        + 'void main(){float inv=1.0-uReveal;vec2 noise=vec2(hash(gl_FragCoord.xy+uTime),hash(gl_FragCoord.yx-uTime));'
        + 'vec2 base=vUv;float ripple=min(0.15,abs(uScroll)*0.5);base.x-=(vUv.x-0.5)*(1.0-sin(vUv.y*3.1415926))*ripple;'
        + 'base+=vec2(uSide*inv*0.085,inv*0.045);float revealZoom=mix(0.72,1.0,uReveal);float hoverZoom=mix(1.0,0.985,uHover);'
        + 'vec2 uv=0.5+(base-0.5)*uSpan*revealZoom*hoverZoom;vec2 view=uMouse*vec2(1.0,-1.0)*uHover*0.075;'
        + 'vec2 puv=uv-view*0.045;puv.y=1.0-puv.y;float focus=length(uMouse)*uHover;float blur=mix(0.0,0.004,uHover)*linstep(0.04,0.35,focus);vec3 col=sampleColor(puv,blur,noise);'
        + 'float luma=dot(col,vec3(0.299,0.587,0.114));col=mix(vec3(luma),col,mix(0.78,1.03,uReveal));'
        + 'col*=1.0-uHover*0.08;col+=uHover*0.018;float a=roundedAlpha(vUv)*uAlpha;gl_FragColor=vec4(col,a);}';

    const states = new Map();   // card -> state
    const active = new Set();
    const queued = new Set();
    const uploadQueue = [];
    let canvas = null, gl = null, prog = null, buf = null, locs = null;
    let io = null, rafId = 0, disabled = false;
    let queueScheduled = false;
    let lastY = window.scrollY || window.pageYOffset || 0;
    let lastT = 0;
    let lastScrollAt = 0;
    let scrollListenerReady = false;
    let scrollMotion = 0;

    const clamp = (v, min, max) => Math.max(min, Math.min(max, v));
    const collect = (root, selector, selfSelector = selector) => {
        const base = root instanceof Element ? root : document;
        const nodes = [];
        if (base.matches?.(selfSelector)) nodes.push(base);
        nodes.push(...base.querySelectorAll(selector));
        return nodes;
    };
    const requestIdle = window.requestIdleCallback
        ? (fn) => window.requestIdleCallback(fn, { timeout: 3000 })
        : (fn) => setTimeout(() => fn({ timeRemaining: () => 8 }), 120);

    const compile = (type, src) => {
        const sh = gl.createShader(type);
        gl.shaderSource(sh, src);
        gl.compileShader(sh);
        if (!gl.getShaderParameter(sh, gl.COMPILE_STATUS)) throw new Error(gl.getShaderInfoLog(sh) || 'shader compile failed');
        return sh;
    };

    const ensureStage = () => {
        if (disabled) return false;
        if (gl) return true;
        const host = document.querySelector('main') || document.body;
        canvas = document.createElement('canvas');
        canvas.className = 'smoe-stage-canvas';
        host.prepend(canvas);
        gl = canvas.getContext('webgl', { alpha: true, antialias: true, depth: false, premultipliedAlpha: false })
            || canvas.getContext('experimental-webgl');
        if (!gl) { disabled = true; canvas.remove(); return false; }
        try {
            prog = gl.createProgram();
            gl.attachShader(prog, compile(gl.VERTEX_SHADER, VERT));
            gl.attachShader(prog, compile(gl.FRAGMENT_SHADER, FRAG));
            gl.linkProgram(prog);
            if (!gl.getProgramParameter(prog, gl.LINK_STATUS)) throw new Error(gl.getProgramInfoLog(prog) || 'program link failed');
        } catch (e) {
            disabled = true;
            canvas.remove();
            return false;
        }
        gl.useProgram(prog);
        buf = gl.createBuffer();
        gl.bindBuffer(gl.ARRAY_BUFFER, buf);
        gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 1, -1, -1, 1, 1, 1]), gl.STATIC_DRAW);
        const p = gl.getAttribLocation(prog, 'p');
        gl.enableVertexAttribArray(p);
        gl.vertexAttribPointer(p, 2, gl.FLOAT, false, 0, 0);
        locs = {};
        ['uColor', 'uRect', 'uMouse', 'uSpan', 'uReveal', 'uHover', 'uSide', 'uScroll', 'uTime', 'uSize', 'uRadius', 'uAlpha'].forEach(n => locs[n] = gl.getUniformLocation(prog, n));
        gl.uniform1i(locs.uColor, 0);
        gl.disable(gl.DEPTH_TEST);
        gl.enable(gl.BLEND);
        gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
        gl.clearColor(0, 0, 0, 0);
        canvas.addEventListener('webglcontextlost', (e) => {
            e.preventDefault();
            disabled = true;
            states.forEach(st => { st.card.dataset.stageReady = '0'; });
            stopLoop();
        }, false);
        return true;
    };

    const resizeStage = () => {
        if (!gl) return;
        const dpr = Math.min(window.devicePixelRatio || 1, 1.5);
        const w = Math.max(1, Math.round(window.innerWidth * dpr));
        const h = Math.max(1, Math.round(window.innerHeight * dpr));
        if (canvas.width !== w || canvas.height !== h) {
            canvas.width = w;
            canvas.height = h;
            gl.viewport(0, 0, w, h);
        }
    };

    const textureSource = async (st) => {
        const img = st.img;
        const rect = st.card.getBoundingClientRect();
        const dpr = Math.min(window.devicePixelRatio || 1, 1.5);
        const displayEdge = Math.max(rect.width || 0, rect.height || 0) * dpr * 1.25;
        const targetEdge = Math.min(MAX_TEXTURE_EDGE, Math.max(720, Math.ceil(displayEdge || MAX_TEXTURE_EDGE)));
        const sourceEdge = Math.max(img.naturalWidth, img.naturalHeight);
        try { await img.decode?.(); } catch (e) { /* decoded enough or failed image fallback */ }
        if (sourceEdge <= targetEdge || !window.createImageBitmap) return img;
        const scale = targetEdge / sourceEdge;
        try {
            return await createImageBitmap(img, {
                resizeWidth: Math.max(1, Math.round(img.naturalWidth * scale)),
                resizeHeight: Math.max(1, Math.round(img.naturalHeight * scale)),
                resizeQuality: 'medium',
            });
        } catch (e) {
            return img;
        }
    };

    const texFromSource = (source) => {
        const tex = gl.createTexture();
        gl.activeTexture(gl.TEXTURE0);
        gl.bindTexture(gl.TEXTURE_2D, tex);
        gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, false);
        gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, source);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
        gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);
        return tex;
    };

    const updateScrollMotion = (now) => {
        if (!lastT) lastT = now;
        const y = window.scrollY || window.pageYOffset || 0;
        const dt = Math.max(16, now - lastT);
        const frameDelta = (y - lastY) / dt * 16.67;
        scrollMotion += (clamp(frameDelta / 90, -1, 1) - scrollMotion) * 0.12;
        if (Math.abs(scrollMotion) < 0.0005) scrollMotion = 0;
        lastY = y;
        lastT = now;
    };

    const upload = async (st) => {
        if (!ensureStage() || st.ready || st.uploading || !st.card.isConnected) return;
        const img = st.img;
        if (!(img.complete && img.naturalWidth)) return;
        st.uploading = true;
        let source = null;
        try {
            source = await textureSource(st);
            if (!st.card.isConnected || st.ready || !ensureStage()) return;
            st.tex = texFromSource(source);
            st.cw = source.width || img.naturalWidth;
            st.ch = source.height || img.naturalHeight;
            st.ready = true;
            st.card.dataset.stageReady = '1';
            if (active.has(st)) startLoop();
        } catch (e) {
            st.card.dataset.stageReady = '0';
        } finally {
            if (source && source !== img && source.close) source.close();
            st.uploading = false;
        }
    };

    const render = (st, now) => {
        if (!st.ready) return;
        const r = st.card.getBoundingClientRect();
        if (!r.width || !r.height || r.bottom < -200 || r.top > window.innerHeight + 200) return;
        if (Math.abs(r.width - st.w) > 0.5 || Math.abs(r.height - st.h) > 0.5) {
            st.w = r.width; st.h = r.height;
            const cAsp = r.width / r.height, iAsp = st.cw / st.ch;
            st.span = iAsp > cAsp ? [cAsp / iAsp, 1] : [1, iAsp / cAsp];
        }
        const vh = window.innerHeight || 1;
        // 入场曲线跟随视口进入进度。canvas 会盖住 <img>，所以图片内缩必须在 shader 里做。
        // 不锁定完成态：向上滚回去后再向下滚，同一张卡也会重新 zoom-settle。
        const travel = Math.max(1, vh * 0.42 + r.height * 0.28);
        const p = clamp((vh - r.top) / travel, 0, 1);
        const target = 1 - Math.pow(1 - p, 3);
        st.reveal += (target - st.reveal) * 0.22;
        st.scroll += (scrollMotion - st.scroll) * 0.18;
        st.tmx += (st.mx - st.tmx) * 0.16;
        st.tmy += (st.my - st.tmy) * 0.16;
        st.hov += (st.hovT - st.hov) * 0.12;
        const vw = window.innerWidth || 1, vh2 = window.innerHeight || 1;
        const left = r.left / vw * 2 - 1;
        const right = r.right / vw * 2 - 1;
        const bottom = 1 - r.bottom / vh2 * 2;
        const top = 1 - r.top / vh2 * 2;
        gl.activeTexture(gl.TEXTURE0);
        gl.bindTexture(gl.TEXTURE_2D, st.tex);
        gl.uniform4f(locs.uRect, left, bottom, right, top);
        gl.uniform2f(locs.uMouse, st.tmx, st.tmy);
        gl.uniform2f(locs.uSpan, st.span[0], st.span[1]);
        gl.uniform1f(locs.uReveal, st.reveal);
        gl.uniform1f(locs.uHover, st.hov);
        gl.uniform1f(locs.uSide, st.side);
        gl.uniform1f(locs.uScroll, st.scroll);
        gl.uniform1f(locs.uTime, now);
        gl.uniform2f(locs.uSize, r.width, r.height);
        gl.uniform1f(locs.uRadius, 15);
        gl.uniform1f(locs.uAlpha, clamp(st.reveal * 1.45, 0, 1));
        gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
    };

    const loop = (now) => {
        if (!ensureStage()) return;
        updateScrollMotion(now);
        resizeStage();
        gl.useProgram(prog);
        gl.bindBuffer(gl.ARRAY_BUFFER, buf);
        gl.clear(gl.COLOR_BUFFER_BIT);
        active.forEach(st => render(st, now));
        rafId = requestAnimationFrame(loop);
    };
    const startLoop = () => { if (!rafId) rafId = requestAnimationFrame(loop); };
    const stopLoop = () => {
        if (rafId) { cancelAnimationFrame(rafId); rafId = 0; }
        if (gl) { resizeStage(); gl.clear(gl.COLOR_BUFFER_BIT); }
    };

    const scheduleQueue = () => {
        if (queueScheduled) return;
        queueScheduled = true;
        requestIdle(() => {
            queueScheduled = false;
            if (performance.now() - lastScrollAt < 220) {
                scheduleQueue();
                return;
            }
            const st = uploadQueue.shift();
            if (!st) return;
            queued.delete(st);
            upload(st).finally(() => {
                if (uploadQueue.length) scheduleQueue();
            });
        });
    };

    const enqueueUpload = (st, immediate = false) => {
        if (!st || st.ready || st.uploading || queued.has(st) || disabled) return;
        if (!(st.img.complete && st.img.naturalWidth)) return;
        if (immediate) {
            upload(st);
            return;
        }
        queued.add(st);
        uploadQueue.push(st);
        scheduleQueue();
    };

    const register = (card) => {
        if (states.has(card) || card.dataset.gl === '1') return;
        const img = card.querySelector('img');
        if (!img) return;
        const li = card.closest('li');
        const side = li && li.matches(':nth-child(even)') ? -1 : 1;
        const st = { card, img, side, ready: false, scroll: 0, mx: 0, my: 0, tmx: 0, tmy: 0, hov: 0, hovT: 0, reveal: 0, w: 0, h: 0, span: [1, 1], cw: 1, ch: 1 };
        states.set(card, st);
        card.dataset.gl = '1';
        io.observe(card);
        if (!(img.complete && img.naturalWidth)) {
            img.addEventListener('load', () => {
                if (card.dataset.glVisible === '1') enqueueUpload(st, true);
            }, { once: true });
            img.addEventListener('error', () => states.delete(card), { once: true });
        }
        card.addEventListener('pointermove', (e) => {
            const r = card.getBoundingClientRect();
            st.mx = (e.clientX - r.left) / r.width - 0.5;
            st.my = (e.clientY - r.top) / r.height - 0.5;
        });
        card.addEventListener('pointerenter', () => { st.hovT = 1; });
        card.addEventListener('pointerleave', () => { st.hovT = 0; st.mx = 0; st.my = 0; });
    };

    const observe = (root = document) => {
        if (!scrollListenerReady) {
            scrollListenerReady = true;
            window.addEventListener('scroll', () => { lastScrollAt = performance.now(); }, { passive: true });
        }
        if (!io) {
            io = new IntersectionObserver((ents) => {
                ents.forEach(en => {
                    const st = states.get(en.target);
                    if (!st) return;
                    en.target.dataset.glVisible = en.isIntersecting ? '1' : '0';
                    if (en.isIntersecting) {
                        active.add(st);
                        enqueueUpload(st, true);
                        startLoop();
                    } else {
                        active.delete(st);
                        if (!active.size) stopLoop();
                    }
                });
            }, { rootMargin: '600px 0px' });
        }
        collect(root, '.card-cover').forEach(card => {
            if (!card.closest('.issue-list')) return;
            register(card);
        });
    };

    return { observe };
})();

document.addEventListener('alpine:init', () => {
    Alpine.data('smoe', () => {
        const resetCopyState = (pre) => {
            delete pre.dataset.copied;
            delete pre.dataset.error;
            pre.dataset.copyTip = 'Click to copy';
        };

        const fallbackCopyText = (text) => {
            const textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.setAttribute('readonly', '');
            textarea.style.position = 'fixed';
            textarea.style.opacity = '0';
            textarea.style.pointerEvents = 'none';
            document.body.append(textarea);
            textarea.focus();
            textarea.select();
            const copied = document.execCommand('copy');
            textarea.remove();
            if (!copied) throw new Error('copy failed');
        };

        const copyText = async (text) => {
            if (navigator.clipboard?.writeText && window.isSecureContext) {
                return navigator.clipboard.writeText(text);
            }
            fallbackCopyText(text);
        };

        const handleCodeCopy = async (pre) => {
            const code = pre.querySelector('code');
            const text = (code ? code.innerText : pre.innerText).replace(/\n$/, '');

            clearTimeout(pre._copyTimer);
            try {
                await copyText(text);
                delete pre.dataset.error;
                pre.dataset.copied = '';
                pre.dataset.copyTip = 'Copied';
            } catch {
                delete pre.dataset.copied;
                pre.dataset.error = '';
                pre.dataset.copyTip = 'Copy failed';
            }

            pre._copyTimer = setTimeout(() => resetCopyState(pre), 1600);
        };

        const enhanceCodeBlocks = (root = document) => {
            root.querySelectorAll('.markdown-body pre').forEach(pre => {
                if (pre.dataset.copyReady === '1') return;
                pre.dataset.copyReady = '1';
                pre.dataset.copyTip = 'Click to copy';
                pre.setAttribute('aria-label', 'Click to copy code');
                pre.addEventListener('click', async (event) => {
                    if (window.getSelection()?.toString()) return;
                    event.stopPropagation();
                    await handleCodeCopy(pre);
                });
                pre.addEventListener('keydown', async (event) => {
                    if (event.key !== 'Enter' && event.key !== ' ') return;
                    event.preventDefault();
                    await handleCodeCopy(pre);
                });
            });
        };

        const collect = (root, selector, selfSelector = selector) => {
            const base = root instanceof Element || root instanceof DocumentFragment ? root : document;
            const nodes = [];
            if (base instanceof Element && base.matches?.(selfSelector)) nodes.push(base);
            nodes.push(...base.querySelectorAll(selector));
            return nodes;
        };

        let titleReelObserver;
        const enhanceTitleReels = (root = document) => {
            if (!titleReelObserver) {
                titleReelObserver = new IntersectionObserver((entries) => {
                    entries.forEach(entry => {
                        if (entry.isIntersecting) {
                            entry.target.dataset.titleIn = '1';
                            titleReelObserver.unobserve(entry.target);
                        }
                    });
                }, { rootMargin: '0px 0px -12% 0px', threshold: 0.18 });
            }

            collect(root, '.card-body h3 > a').forEach(anchor => {
                if (!anchor.closest('.issue-list')) return;
                if (anchor.dataset.reelReady === '1') return;
                anchor.dataset.reelReady = '1';
                const text = anchor.textContent;
                anchor.setAttribute('aria-label', text);
                anchor.textContent = '';
                Array.from(text).forEach((char, index) => {
                    const wrap = document.createElement('span');
                    wrap.className = 'title-reel-char';
                    wrap.setAttribute('aria-hidden', 'true');
                    wrap.style.setProperty('--i', index);
                    if (/\s/.test(char)) {
                        wrap.classList.add('is-space');
                        anchor.append(wrap);
                        return;
                    }
                    const track = document.createElement('span');
                    track.className = 'title-reel-track';
                    for (let i = 0; i < 4; i++) {
                        const item = document.createElement('span');
                        item.textContent = char;
                        track.append(item);
                    }
                    wrap.append(track);
                    anchor.append(wrap);
                });
                const item = anchor.closest('.issue-list > li');
                if (item) {
                    const rect = item.getBoundingClientRect();
                    if (rect.top < innerHeight * 0.88 && rect.bottom > innerHeight * 0.08) {
                        item.dataset.titleIn = '1';
                    } else {
                        titleReelObserver.observe(item);
                    }
                }
            });
        };

        return {
            hasMore: true,
            pageNum: (() => {
                const match = window.location.pathname.match(/\/page\/(\d+)/);
                return match ? parseInt(match[1]) + 1 : 2;
            })(),
            darkMode: localStorage.getItem('theme') === 'dark',

            lightbox(e) {
                if (e.target.tagName !== 'IMG' || e.target.closest('a, dialog')) return;
                const dialog = e.currentTarget.querySelector('dialog');
                dialog.querySelector('img').src = e.target.src;
                dialog.showModal();
            },

            async likePost(el) {
                if ('liked' in el.dataset) return;
                const article = el.closest('article');
                const cid = article.dataset.id;
                const res = await fetch(`/archives/${cid}/like`, { method: 'POST' });
                const data = await res.json();
                if (data.liked) {
                    el.dataset.liked = '';
                    localStorage.setItem('liked:' + cid, '1');
                    smoeBurst(el);
                    const num = article.querySelector('[data-like-count]');
                    if (num) num.textContent = parseInt(num.textContent) + 1;
                }
            },

            toggleDark(e) {
                const btn = e.currentTarget;
                const rect = btn.getBoundingClientRect();
                const x = rect.left + rect.width / 2;
                const y = rect.top + rect.height / 2;
                document.documentElement.style.setProperty('--tx', x + 'px');
                document.documentElement.style.setProperty('--ty', y + 'px');

                const apply = () => {
                    this.darkMode = !this.darkMode;
                    document.documentElement.dataset.theme = this.darkMode ? 'dark' : '';
                    localStorage.setItem('theme', this.darkMode ? 'dark' : 'light');
                };

                if (!document.startViewTransition) return apply();
                // 标记主题切换，让圆形扩散动画只作用于这次同文档过渡，
                // 不污染跨文档导航的 root 过渡
                document.documentElement.dataset.vt = 'theme';
                document.startViewTransition(apply).finished.finally(() => {
                    delete document.documentElement.dataset.vt;
                });
            },

            goBack() {
                history.length > 1 ? history.back() : (location.href = '/');
            },

            // 封面取色 → 整页换装（--c-accent 全局跟随）
            theming(img) {
                const apply = () => {
                    try {
                        const swatches = new Vibrant(img).swatches();
                        const pick = (...names) => names.map(n => swatches[n]).find(Boolean);
                        const forLight = pick('DarkVibrant', 'Vibrant', 'DarkMuted');
                        const forDark = pick('LightVibrant', 'Vibrant', 'LightMuted');
                        if (!forLight || !forDark) return;
                        const s = document.documentElement.style;
                        s.setProperty('--accent-light', forLight.getHex());
                        s.setProperty('--accent-dark', forDark.getHex());
                        document.documentElement.classList.add('has-palette');
                    } catch { /* 取色失败则保持默认印章红 */ }
                };
                img.complete && img.naturalWidth ? apply() : img.addEventListener('load', apply, { once: true });
            },

            init() {
                if (this.darkMode) document.documentElement.dataset.theme = 'dark';
                enhanceCodeBlocks(document);
                enhanceTitleReels(document);
                window.$ = (selector) => document.querySelector(selector);
                this.lenis = smoeSmooth();
                smoeStage.observe(document);
            },

            async ajaxNextPage(e) {
                const button = e.currentTarget;
                if (button.disabled || !this.hasMore) return;
                button.disabled = true;
                const text = button.textContent;
                button.textContent = '加载中...';
                try {
                    const res = await fetch(`/page/${this.pageNum}`);
                    this.hasMore = res.headers.get('X-Has-More') === 'true';
                    const data = await res.text();
                    if (data.trim()) {
                        const main = $('main');
                        let list = $('main .issue-list');
                        if (!list && main) {
                            list = document.createElement('ol');
                            list.className = 'issue-list';
                            main.insertBefore(list, main.querySelector('.load-more'));
                        }
                        if (list) {
                            const tpl = document.createElement('template');
                            tpl.innerHTML = data.trim();
                            const items = [...tpl.content.children];
                            list.append(...items);
                            items.forEach(item => {
                                enhanceTitleReels(item);
                                smoeStage.observe(item);
                            });
                        }
                    }
                    this.pageNum++;
                } finally {
                    button.disabled = false;
                    button.textContent = text;
                }
            },

            bgmPlayer: {
                playing: false,
                'x-ref': 'bgmPlayer',
                '@timeupdate'(e) {
                    this.bgmPlayer.playing = !e.target.paused;
                    e.target.parentElement.style.setProperty('--play-progress', e.target.currentTime / e.target.duration * 100 + '%');
                },
                '@ended'() { console.log('bgm complete'); },
                Button: {
                    '@click'() {
                        this.$refs.bgmPlayer.paused ? this.$refs.bgmPlayer.play() : this.$refs.bgmPlayer.pause();
                    },
                },
            },

            comment: {
                parent: 0,
                cid: 0,
                Reply: {
                    '@click'() {
                        let li = event.target.closest('li');
                        li.insertAdjacentElement('afterend', this.$refs.comment);
                        this.comment.parent = li.id;
                    },
                },
                CancelReply: {
                    '@click'() {
                        event.target.closest('footer').insertAdjacentElement('afterbegin', this.$refs.comment);
                        this.comment.parent = 0;
                    },
                },
                'x-ref': 'comment',
                'x-init'() {
                    this.$refs.comment.action = `${window.location}/comment`;
                    this.comment.cid = window.location.pathname.split('/').at(-1);
                },
                '@submit.prevent'(e) {
                    let form = e.target;
                    fetch(form.action, {
                        method: 'POST',
                        body: new FormData(form)
                    }).then(r => {
                        if (r.status === 200) {
                            alert('Comments will be published after review. Thank you');
                            form.reset();
                        }
                    });
                },
            },
        };
    })
})
