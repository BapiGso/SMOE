/*
 * Smoe
 * @author Smoe
 * @url https://smoe.cc
 */
gsap.registerPlugin(ScrollTrigger, SplitText);

// 开场推门（#door）拉开后整块移除，免得花在屏幕外继续转
{
    const door = document.getElementById('door');
    door?.addEventListener('animationend', (e) => e.animationName === 'door-l' && door.remove());
}

document.addEventListener('alpine:init', () => {
    Alpine.data('smoe', () => {
        const preview = Object.assign(document.createElement('aside'));
        // 系统开了“减少动态效果”时不做入场和视差，文章层直接切换
        const motion = !matchMedia('(prefers-reduced-motion: reduce)').matches;
        // 整页加载时有约 3 秒的开场推门（#door，2 秒时开始拉开），入场动画等它拉开再开始
        const curtain = 2.4;

        // 中文没有空格，SplitText 默认会把整段标题当成一个不能换行的词；先按词切开，再用不可见分隔符交给它。
        // 标点不单独成词（右标点粘前一个词，左括号/引号粘后一个词），否则换行时会出现行首的“！”“）”
        const sep = String.fromCharCode(0x2063);
        const segmenter = new Intl.Segmenter('zh', { granularity: 'word' });
        const glue = (a, b) => /[\p{Ps}\p{Pi}]$/u.test(a) || (/\S$/.test(a) && /^[\p{Pe}\p{Pf}\p{Po}\p{Pd}]/u.test(b));
        const splitChars = (el) => SplitText.create(el, {
            type: 'chars',
            smartWrap: true,
            prepareText: (text) => Array.from(segmenter.segment(text), s => s.segment)
                .reduce((out, s, i, all) => out + (i && !glue(all[i - 1], s) ? sep : '') + s, ''),
            wordDelimiter: { delimiter: sep, replaceWith: '' },
        });

        const revealTitle = (el, delay) => {
            if (!el || !motion) return;
            gsap.from(splitChars(el).chars, {
                yPercent: 60, autoAlpha: 0, duration: 0.6, ease: 'back.out(1.7)', stagger: 0.03, delay,
            });
        };

        // 卡片滚动进视口时依次浮现；“加载更多”拼进来的卡片同样走这里
        const revealCards = (cards) => {
            if (!cards.length || !motion) return;
            gsap.set(cards, { autoAlpha: 0, y: 40 });
            ScrollTrigger.batch(cards, {
                start: 'top 90%',
                once: true,
                onEnter: (batch) => gsap.to(batch, { autoAlpha: 1, y: 0, duration: 0.6, ease: 'power2.out', stagger: 0.12 }),
            });
        };

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

        // overwrite: 关到一半又打开时，杀掉关闭动画，它的 onComplete 就不会把新打开的层移除
        const openPreview = () => {
            document.body.prepend(preview);
            document.documentElement.className = '';
            document.documentElement.style.overflow = 'hidden';
            gsap.fromTo(preview, { xPercent: 100 }, { xPercent: 0, duration: motion ? 0.5 : 0, ease: 'power2.out', overwrite: true });
        };

        const closePreview = () => {
            document.documentElement.style.overflow = '';
            gsap.to(preview, {
                xPercent: 100, duration: motion ? 0.5 : 0, ease: 'power2.out', overwrite: true,
                onComplete: () => preview.remove(),
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
                    gsap.to(document.body, {
                        xPercent: 100, duration: motion ? 0.5 : 0, ease: 'power2.out',
                        onComplete: () => location.href = '/',
                    });
                }
            },

            // 首图跟随鼠标反向偏移，最多 ±4vw/vh（图片宽高就是 100vw/vh）；quickTo 追到目标后自己停，不再每帧空转
            parallax(el) {
                if (!motion) return;
                const img = el.firstElementChild;
                const xTo = gsap.quickTo(img, 'xPercent', { duration: 0.8, ease: 'power3' });
                const yTo = gsap.quickTo(img, 'yPercent', { duration: 0.8, ease: 'power3' });
                window.addEventListener('mousemove', (e) => {
                    xTo((e.clientX / window.innerWidth - 0.5) * -8);
                    yTo((e.clientY / window.innerHeight - 0.5) * -8);
                }, { passive: true });
            },

            init() {
                if (this.darkMode) document.documentElement.dataset.theme = 'dark';
                enhanceCodeBlocks(document);
                window.$ = (selector) => document.querySelector(selector);
                // 首页：最新一篇的标题逐字浮现，日期和摘要随后淡入；文章页直接打开时标题同样逐字浮现
                const hero = $('main > div');
                if (hero && motion) {
                    revealTitle(hero.querySelector('h2 > a'), curtain);
                    gsap.from(hero.querySelectorAll(':scope > p, :scope > div'), {
                        autoAlpha: 0, y: 20, duration: 0.8, ease: 'power2.out', stagger: 0.2, delay: curtain + 0.5,
                    });
                }
                revealTitle($('body > article > h1'), curtain);
                revealCards(document.querySelectorAll('main > ol > li'));
                // 窄屏下卡片高度随懒加载的封面变化，图片加载完（load 不冒泡，用捕获）重新计算触发位置
                const refresh = gsap.delayedCall(0.3, () => ScrollTrigger.refresh()).pause();
                $('main')?.addEventListener('load', () => refresh.restart(true), true);
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
                revealTitle(preview.querySelector('article > h1'), 0.3);
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
                    revealCards(cards);
                    this.hasMore = doc.querySelector('main > button[data-more]')?.dataset.more === 'true';
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
