/*
 * Smoe
 * @author Smoe
 * @url https://smoe.cc
 */
document.addEventListener('alpine:init', () => {
    Alpine.data('smoe', () => ({
        init() {
            window.$ = (selector) => document.querySelector(selector);//封装一个简易的jquery选择器
            window.preview = Object.assign(document.createElement("div"), { id: "preview" });
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
            document.querySelectorAll('.icon-menu .line').forEach(function (icon) {
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


let scrollWidth=()=>{
        if (!('ontouchstart' in window)) {
            let st = window.scrollY||document.documentElement.scrollTop ;
            if (document.getElementById('preview')){
                st= document.getElementById('preview').scrollTop;
            }
            let ct = document.getElementsByTagName('article').offsetHeight;
            if (st > ct) {
                st = ct;
            }
            return ((50 + st) / ct * 100) + '%';
        }
    };



let Diaspora = {
    SWLoader:function (){
      if(document.getElementById("loader")){
          document.getElementById("loader").remove()
      }else{
          let l=Object.assign(document.createElement("div"), { id: "loader" });
          document.body.insertAdjacentElement("afterbegin",l)
      }
    },

    // 传ajax的url和响应成功后需要执行的函数
    AJAX:async function (url,func,errfunc){
        fetch(url)
            .then(response => response.text())
            .then(html => {
                    func(html)
            })
            .catch(error => {
                errfunc()
                // 处理错误
                console.error('请求失败', error);
                debugger;
                window.location.href = url;
            });
    },

    //传进来元素和flag状态
    SingleLoader:function (tag,flag) {
        let id = tag.getAttribute('data-id') || 0,
            url = tag.getAttribute('href'),
            title = tag.getAttribute('title') || tag.innerText || tag.textContent; // 根据需要获取文本内容
        let state = { d: id, t: title, u: url };

        Diaspora.AJAX(url,function (data) {

            switch (flag) {
                case 'push':
                    debugger
                    history.pushState(state, title, url)
                    break;
                //评论翻页的时候会用这个
                case 'replace':
                    history.replaceState(state, title, url)
                    break;
            }
        })
    },

    PS: function () {
        if (!(window.history && history.pushState)) return;

        history.replaceState({ u: Home, t: document.title }, document.title, Home)

        window.addEventListener('popstate', function (e) {
            let state = e.state;
            if (!state) return;
            document.title = state.t;

            if (state.u === Home) {
                this.preview.style.transform="translateX(100%)"
                document.getElementById('container').style.display = 'block';
                // window.scrollTo(0, parseInt(document.getElementById('container').dataset.scroll));
            } else {//后退之后又前进
                document.body.insertAdjacentElement("afterbegin", preview)
                //todo 后退后前进滚动条会跳动
                // window.scrollTo(0, parseInt(document.getElementById('container').dataset.scroll));
                this.preview.style.transform = "unset";
                (async function () {
                    await new Promise(resolve => setTimeout(resolve, 500));
                    document.querySelector("#container").style.display = "none"
                })()
            }

        })
    },

    player: function (id) {
        let au = document.getElementById('audio')
        let playButton = document.querySelector('.icon-play');
        let playbackBar = document.querySelector('.playbackbar');

        if(!au){
            return
        }
        // 检查
        // if (!au.length()) {
        //     //todo 颜色变暗
        //     playButton.style.cursor = 'not-allowed';
        //     return
        // }
        //自动播放，暂时关闭
        // if (p.eq(0).data("autoplay") === false) {
        //     p[0].play();
        // }
        au.addEventListener('timeupdate', function() {
            let progress = (au.currentTime / au.duration) * 100;
            playbackBar.style.width = progress + '%';
        });

        au.addEventListener('ended', function() {
            playButton.classList.remove('icon-pause');
            playButton.classList.add('icon-play');
        });

        au.addEventListener('playing', function() {
            playButton.classList.remove('icon-play');
            playButton.classList.add('icon-pause');
        });
    },

    //给loader添加动画用的
    loading: function () {
        $('#loader').removeClass().addClass('loader890').show()
    },

    //给loader移除动画用的
    loaded: function () {
        $('#loader').removeClass().hide()
    },

    SubmitComment:function (e) {
        if (e.preventDefault) e.preventDefault();
        // 获取表单元素
        const form = document.querySelector('form');

        // 设置表单action


        // 阻止表单默认提交行为
        return false;
    },

    ReplyComment:function () {
        let commentForm = document.getElementById("comment-form")
        const buttongroup = commentForm.querySelector("#buttongroup")
        const cancelButton = document.createElement('button');
        cancelButton.className = "cancelButton";
        cancelButton.innerText = "取消回复";
        cancelButton.addEventListener('click', function () {
            const commentWrap = document.querySelector(".comment-wrap")
            commentWrap.insertAdjacentElement("afterbegin", commentForm)
            cancelButton.remove()
        });
        if (!buttongroup.querySelector(".cancelButton")) {
            buttongroup.insertAdjacentElement("afterbegin", cancelButton);
        }

        commentForm.action = new URL(window.location.href).pathname + "/comment"
        // 获取当前点击的按钮
        const replyButton = event.target;
        // 获取按钮的父节点的父节点
        const grandParentNode = replyButton.parentNode.parentNode;
        // 将表单插入到按钮的父节点的父节点后面
        grandParentNode.insertAdjacentElement("afterend", commentForm);
},

    scroller: function (){
        let scrollbar = document.querySelector('.scrollbar');
        let subtitle = document.querySelector('.subtitle');
        let article = document.getElementsByTagName('article');

        if (scrollbar && !('ontouchstart' in window)) {
            let st = window.scrollY||document.documentElement.scrollTop ;
            if (document.getElementById('preview')){
                st= document.getElementById('preview').scrollTop;
            }
            let ct = article.offsetHeight;
            if (st > ct) {
                st = ct;
            }
            scrollbar.style.width = ((50 + st) / ct * 100) + '%';

            if (st > 80 && window.innerWidth > 800) {
                subtitle.style.visibility = 'visible';
            } else {
                subtitle.style.visibility = 'hidden';
            }
        }
    },

    init:function () {
        if (('ontouchstart' in window)) {
            $('body').addClass('touch')
        }
        if(document.getElementById("container")){
            Diaspora.PS()
        }else {
            window.addEventListener('popstate', function (e) {
                if (e.state) location.href = e.state.u;
            })
            document.querySelector(".logo-min").href="/";
        }

    }
}