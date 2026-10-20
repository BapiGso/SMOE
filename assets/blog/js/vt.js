/*
 * 跨文档 View Transitions：首页封面 ⇄ 文章头图 共享形变 + 标题逐字洇墨
 * 必须在 <head> 同步加载 —— pagereveal 在新页面首帧前触发，放在 body 末尾会错过。
 * 文章页头图的命名写在 CSS 里，这里负责首页一侧的动态命名、返回方向、标题拆字。
 */
(() => {
    if (!('onpagereveal' in window)) return;

    const root = document.documentElement;
    const named = [];
    const name = (el, n) => {
        if (!el) return;
        el.style.viewTransitionName = n;
        named.push(el);
    };
    const reset = () => {
        named.splice(0).forEach(el => { el.style.viewTransitionName = ''; });
        delete root.dataset.vtCover;
    };
    const cidOf = (url) => url && new URL(url).pathname.match(/^\/archives\/(\d+)\/?$/)?.[1];
    const inView = (el) => {
        const r = el.getBoundingClientRect();
        return r.width > 0 && r.bottom > 0 && r.top < innerHeight;
    };

    // 首页：给文章对应的封面命名。卷首文章对应整屏封面，其余对应列表卡片的图。
    // 叠在封面上的元素（卡片纸、卷首标题、导航）也单独命名，否则会被飞起的封面瞬间盖住。
    const bindHome = (cid) => {
        const el = cid && document.querySelector(`[data-vt="${cid}"]`);
        if (!el) return false;
        if (el.matches('.hero-cover')) {
            if (!inView(el)) return false;
            name(el, 'post-cover');
            name(document.querySelector('.hero-block'), 'hero-text');
            name(document.querySelector('main > details > summary'), 'hero-nav');
        } else {
            const img = el.querySelector('img');
            if (!img || !inView(img)) return false;
            name(img, 'post-cover');
            name(el.closest('li')?.querySelector('.card-body'), 'card-paper');
        }
        root.dataset.vtCover = '';   // 过渡期间显示原生 <img>、隐藏 WebGL 画布
        return true;
    };

    // 文章页：标题拆成逐字（西文按词）的 span，由 CSS 依次洇墨浮现；刊记排在最后一个字之后
    const inkTitle = () => {
        const h1 = document.querySelector('body > article > h1');
        if (!h1 || 'inked' in h1.dataset) return;
        const text = h1.textContent.trim();
        const parts = text.match(/[\w'’.-]+|\s+|./gu) || [];
        const count = parts.filter(p => p.trim()).length;
        const step = Math.min(60, 900 / Math.max(1, count));
        h1.dataset.inked = '';
        h1.setAttribute('aria-label', text);
        h1.textContent = '';
        let i = 0;
        parts.forEach(p => {
            if (!p.trim()) return h1.append(' ');
            const s = document.createElement('span');
            s.className = 'ink';
            s.setAttribute('aria-hidden', 'true');
            s.textContent = p;
            s.style.setProperty('--d', Math.round(i++ * step) + 'ms');
            h1.append(s);
        });
        h1.nextElementSibling?.style.setProperty('--ink-end', Math.round(600 + i * step) + 'ms');
        root.dataset.ink = '';
    };

    addEventListener('pageswap', (e) => {
        if (!e.viewTransition) return;
        const hero = document.querySelector('.post-hero img');
        if (hero) {
            // 文章已读到下方、头图不在视口：不参与形变，免得从屏外飞回来
            if (!inView(hero)) name(hero, 'none');
            return;
        }
        bindHome(cidOf(e.activation?.entry?.url));
    });

    addEventListener('pagereveal', (e) => {
        reset();   // 清掉 bfcache 恢复时残留的命名
        const vt = e.viewTransition;
        if (!vt) return;
        const act = window.navigation?.activation;
        if (act?.navigationType === 'traverse' && act.from && act.entry.index < act.from.index) {
            vt.types?.add('back');
        }
        if (bindHome(cidOf(act?.from?.url))) vt.finished.finally(reset);
        inkTitle();
    });
})();
