/*
 * Smoe
 * @author Smoe
 * @url https://smoe.cc
 */
document.addEventListener('alpine:init', () => {
    Alpine.data('smoe', () => {
        const preview = Object.assign(document.createElement('aside'));
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

        const openPreview = () => {
            document.body.insertAdjacentElement('afterbegin', preview);
            requestAnimationFrame(() => requestAnimationFrame(() => {
                document.documentElement.className = '';
                document.documentElement.style.overflow = 'hidden';
                preview.style.transform = 'unset';
            }));
        };

        const closePreview = () => {
            preview.style.transform = 'translateX(100%)';
            document.documentElement.style.overflow = '';
            setTimeout(() => preview.remove(), 500);
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
                document.startViewTransition(apply);
            },

            goBack() {
                if (preview.parentElement) {
                    closePreview();
                    history.back();
                } else {
                    document.body.style.transition = 'transform 0.5s cubic-bezier(0.25, 0.5, 0.5, 0.9)';
                    document.body.style.transform = 'translateX(100%)';
                    setTimeout(() => location.href = '/', 500);
                }
            },

            parallax(el) {
                const img = el.firstElementChild;
                let tx = 0, ty = 0, cx = 0, cy = 0;
                const friction = 0.1;
                const onMove = e => {
                    tx = (e.clientX / window.innerWidth  - 0.5) * -8;
                    ty = (e.clientY / window.innerHeight - 0.5) * -8;
                };
                window.addEventListener('mousemove', onMove, { passive: true });
                const loop = () => {
                    cx += (tx - cx) * friction;
                    cy += (ty - cy) * friction;
                    img.style.transform = `translate3d(${cx.toFixed(2)}vw,${cy.toFixed(2)}vh,0)`;
                    requestAnimationFrame(loop);
                };
                loop();
            },

            init() {
                if (this.darkMode) document.documentElement.dataset.theme = 'dark';
                enhanceCodeBlocks(document);
                window.$ = (selector) => document.querySelector(selector);
                window.onpopstate = (e) => {
                    if (!e.state) return (location.href = location.origin);
                    document.title = e.state.t;
                    //todo /page/2这种情况要特殊判断
                    e.state.u === location.origin ? closePreview() : openPreview();
                };
            },

            async ajaxPost(url) {
                const doc = new DOMParser().parseFromString(await (await fetch(url)).text(), 'text/html');
                preview.innerHTML = doc.body.innerHTML;
                enhanceCodeBlocks(preview);
                history.replaceState({ t: document.title, u: location.origin }, '', location.href);
                history.pushState({ t: doc.title, u: url }, '', url);
                document.title = doc.title;
                this.$refs.nav.open = false;
                openPreview();
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
                    if (data.trim()) $('main > ol').insertAdjacentHTML('beforeend', data);
                    this.pageNum++;
                } finally {
                    button.disabled = false;
                    button.textContent = text;
                }
            },

            vibrant(el) {
                const swatches = new Vibrant(el).swatches();
                $('#vibrant polygon').style.fill = swatches['DarkVibrant'].getHex();
                document.querySelectorAll('.line1, .line2, .line3').forEach(icon => {
                    icon.style.stroke = swatches['Vibrant'].getHex();
                });
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
