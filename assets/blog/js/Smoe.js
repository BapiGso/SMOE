/*
 * Smoe
 * @author Smoe
 * @url https://smoe.cc
 */
document.addEventListener('alpine:init', () => {
    Alpine.data('smoe', () => ({
        init() {
            window.$ = (selector) => document.querySelector(selector);//封装一个简易的jquery选择器
            window.preview = Object.assign(document.createElement("aside"), { id: "preview" });
            window.onpopstate=function (e) {
                if (!e.state) return  location.href=location.origin;//如果没有state说明不是从首页来的，返回首页
                document.title = e.state.t;
                //todo /page/2这种情况要特殊判断
                if (e.state.u === location.origin ) {//如果是后退到主页
                    preview.style.transform = "translateX(100%)";
                    $('html').style.overflow = "";
                    (async function () {
                        await new Promise(resolve => setTimeout(resolve, 500));
                        preview.remove();
                    })()
                } else {//如果是前进到文章或独立页面
                    document.body.insertAdjacentElement("afterbegin", preview);
                    (async function () {
                        await new Promise(resolve => setTimeout(resolve, 95));
                        document.documentElement.className='';
                        document.documentElement.style.overflow = "hidden";
                        preview.style.transform = "unset"
                    })()
                }
            }
        },
        vibrant:function (el) {
            const vibranter = new Vibrant(el);
            const swatches = vibranter.swatches();
            $('#vibrant polygon').style.fill = swatches['DarkVibrant'].getHex();
            document.querySelectorAll('.line1, .line2, .line3').forEach(function (icon) {
                icon.style.stroke = swatches['Vibrant'].getHex();
            });
        },
        hasMore: true,
        pageNum: (() => {
            const match = window.location.pathname.match(/\/page\/(\d+)/);
            return match ? parseInt(match[1]) + 1 : 2;
        })(),
        ajaxPost: function(url){
            fetch(url)
                .then(r => r.text())
                .then(data => {
                    let htmlDoc = new DOMParser().parseFromString(data, 'text/html');
                    preview.innerHTML = htmlDoc.body.innerHTML;
                    history.replaceState({t: document.title, u: location.origin}, '', location.href);//先保存当前记录
                    history.pushState({t: htmlDoc.title, u: url}, '', url);
                    window.dispatchEvent(new PopStateEvent('popstate', {
                        state: {t: htmlDoc.title, u: url}
                    }));//手动触发popstate
                })
        },
        ajaxNextPage: function () {
            fetch(`/page/${this.pageNum}`)
                .then(r => {
                    this.hasMore = r.headers.get('X-Has-More') === 'true';
                    return r.text();
                })
                .then(data => {
                    if (data.trim()) {
                        $('#primary').insertAdjacentHTML('beforeend', data);
                    }
                    this.pageNum++;
                })
        },
        bgmPlayer: {
            playing:false,
            playProgress:0,
            'x-ref':"bgmPlayer",
            '@timeupdate'(e) {
                this.bgmPlayer.playing = !e.target.paused;
                this.bgmPlayer.playProgress = ((e.target.currentTime / e.target.duration) * 100).toFixed(2);
                },
            '@ended'() {console.log("bgm complete");},
            Button: {
                '@click'() {
                    this.$refs.bgmPlayer.paused ? this.$refs.bgmPlayer.play() : this.$refs.bgmPlayer.pause();
                },
            },
        },
        comment:{
            parent: 0,
            cid: 0,
            Reply: {
                '@click'() {
                    let parent = event.target.parentElement.parentElement;
                    parent.insertAdjacentElement("afterend", this.$refs.comment);
                    this.comment.parent = parent.id;
                },
            },
            CancelReply:  {
                '@click'() {
                    $('.comment-wrap').insertAdjacentElement('afterbegin', this.$refs.comment);
                    this.comment.parent = 0;
                },
            },
            'x-ref': 'comment',
            'x-init'(){
                // console.log(this.comment)
                this.$refs.comment.action = `${window.location}/comment`;//设置提交地址
                this.comment.cid = window.location.pathname.split("/")[window.location.pathname.split("/").length - 1];
                // console.log(this.comment.cid)
            },
            '@submit.prevent'(e) {
                let form = e.target;
                fetch(form.action,{
                    method: 'POST',
                    body: new FormData(form)
                }).then(r=>{
                    if (r.status === 200) {
                        alert("Comments will be published after review. Thank you");
                        form.reset();
                    }
                })
            },
        },
    }))
})

