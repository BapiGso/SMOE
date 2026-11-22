/*
 * Smoe
 * @author Smoe
 * @url https://smoe.cc
 */
// 开场推门（#door）拉开后整块移除，免得花在屏幕外继续转
{
    const door = document.getElementById('door');
    door?.addEventListener('animationend', (e) => e.animationName === 'door-l' && door.remove());
}

document.addEventListener('alpine:init', () => {
    Alpine.data('smoe', () => {
        const preview = document.createElement('aside');
        const root = document.documentElement;
        // 深浅色：<meta name="color-scheme"> 是 "light dark"（跟随系统）、"light" 或 "dark"，页面渲染跟着它
        const scheme = document.querySelector('meta[name="color-scheme"]');
        const systemDark = matchMedia('(prefers-color-scheme: dark)');
        const isDark = () => scheme.content === 'dark' || (scheme.content !== 'light' && systemDark.matches);

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

        // 文章层滑入滑出，时长和缓动取 CSS 的 --dur-l、--ease（系统开了“减少动态效果”时时长为 0）。
        // 关到一半又打开时 cancel 掉关闭动画，它的 finished 会 reject，不会把新打开的层移除
        let slide;
        const animateSlide = (open) => {
            slide?.cancel();
            const css = getComputedStyle(root);
            slide = preview.animate([{ translate: '100% 0' }, { translate: '0 0' }], {
                duration: parseFloat(css.getPropertyValue('--dur-l')) * 1000,
                easing: css.getPropertyValue('--ease'),
                direction: open ? 'normal' : 'reverse',
                fill: 'forwards',
            });
            return slide.finished;
        };

        const openPreview = () => {
            document.body.prepend(preview);
            root.className = '';
            root.style.overflow = 'hidden';
            animateSlide(true).catch(() => {});
        };

        const closePreview = () => {
            root.style.overflow = '';
            animateSlide(false).then(() => preview.remove(), () => {});
        };

        return {
            hasMore: true,
            pageNum: (() => {
                const match = window.location.pathname.match(/\/page\/(\d+)/);
                return match ? parseInt(match[1]) + 1 : 2;
            })(),
            darkMode: isDark(),

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

                // 切到与当前所见相反的一边；结果和系统一致时清掉记录，回到跟随系统
                const apply = () => {
                    const target = isDark() ? 'light' : 'dark';
                    const system = systemDark.matches ? 'dark' : 'light';
                    if (target === system) localStorage.removeItem('theme');
                    else localStorage.setItem('theme', target);
                    scheme.content = target === system ? 'light dark' : target;
                    this.darkMode = target === 'dark';
                };

                if (!document.startViewTransition) return apply();
                document.startViewTransition(apply);
            },

            goBack() {
                if (preview.parentElement) {
                    closePreview();
                    history.back();
                } else {
                    location.href = '/';
                }
            },

            // 首图跟随鼠标反向偏移：这里只写鼠标位置，偏移和缓动在 CSS（main > figure > img）
            parallax(el) {
                window.addEventListener('mousemove', (e) => {
                    el.style.setProperty('--mx', e.clientX / window.innerWidth - 0.5);
                    el.style.setProperty('--my', e.clientY / window.innerHeight - 0.5);
                }, { passive: true });
            },

            init() {
                systemDark.addEventListener('change', () => this.darkMode = isDark());
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
                // 只取正文，开场推门和脚本不要跟着进来
                preview.replaceChildren(doc.querySelector('body > article') ?? '');
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
                    // 下一页是完整的静态页面，取出其中的文章卡片和“是否还有下一页”
                    const res = await fetch(`/page/${this.pageNum}`);
                    if (!res.ok) return (this.hasMore = false);
                    const doc = new DOMParser().parseFromString(await res.text(), 'text/html');
                    let list = $('main > ol');
                    if (!list) {
                        list = document.createElement('ol');
                        button.before(list);
                    }
                    const cards = doc.querySelectorAll('main > ol > li');
                    list.append(...cards);
                    this.hasMore = doc.querySelector('main > button[data-more]')?.dataset.more === 'true';
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
                    // 提交地址已写在模板的 action 里
                    this.comment.cid = this.$el.closest('article').dataset.id;
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
