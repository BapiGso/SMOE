/*
 * Smoe
 * @author Smoe
 * @url https://smoe.cc
 */
document.addEventListener('alpine:init', () => {
    Alpine.data('smoe', () => {
        const preview = Object.assign(document.createElement('aside'), { id: 'preview' });

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
            preview.addEventListener('transitionend', () => preview.remove(), { once: true });
        };

        return {
            hasMore: true,
            pageNum: (() => {
                const match = window.location.pathname.match(/\/page\/(\d+)/);
                return match ? parseInt(match[1]) + 1 : 2;
            })(),

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
                preview.innerHTML = doc.documentElement.innerHTML;
                history.replaceState({ t: document.title, u: location.origin }, '', location.href);
                history.pushState({ t: doc.title, u: url }, '', url);
                document.title = doc.title;
                document.getElementById('nav').removeAttribute('open');
                openPreview();
            },

            async ajaxNextPage() {
                const res = await fetch(`/page/${this.pageNum}`);
                this.hasMore = res.headers.get('X-Has-More') === 'true';
                const data = await res.text();
                if (data.trim()) $('#primary').insertAdjacentHTML('beforeend', data);
                this.pageNum++;
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
                playProgress: 0,
                'x-ref': 'bgmPlayer',
                '@timeupdate'(e) {
                    this.bgmPlayer.playing = !e.target.paused;
                    this.bgmPlayer.playProgress = ((e.target.currentTime / e.target.duration) * 100).toFixed(2);
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
                        let parent = event.target.parentElement.parentElement;
                        parent.insertAdjacentElement('afterend', this.$refs.comment);
                        this.comment.parent = parent.id;
                    },
                },
                CancelReply: {
                    '@click'() {
                        $('.comment-wrap').insertAdjacentElement('afterbegin', this.$refs.comment);
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
