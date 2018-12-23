/**
 * demo.js
 * http://www.codrops.com
 *
 * Licensed under the MIT license.
 * http://www.opensource.org/licenses/mit-license.php
 *
 * Copyright 2018, Codrops
 * http://www.codrops.com
 */
// Generate a random number.
const getRandomNumber = (min, max) => (Math.random() * (max - min) + min);
const body = document.body;
//const winsize = {width: window.innerWidth, height: window.innerHeight};


function httpGet(theUrl)
{
    var xmlHttp = new XMLHttpRequest();
    xmlHttp.open( "GET", theUrl, false ); // false for synchronous request
    xmlHttp.send( null );
    return xmlHttp.responseText;
}

function reconstructText(ele) {
    ele.innerHTML = ele.innerHTML.replace(/\&lt;br\&gt;/gi,"\n").replace(/(&lt;([^&gt;]+)&gt;)/gi, "");
    text = ele.innerText;
    return text
}

function deleteSpans(ele) {
    while (ele.firstChild) {
        ele.removeChild(ele.firstChild);
    }
}

class Slide {
    constructor(el) {
        this.DOM = {el: el};
        this.DOM.title = this.DOM.el.querySelector('.slide__title');
        charming(this.DOM.title);
        this.DOM.titleLetters = Array.from(this.DOM.title.querySelectorAll('span'));
        this.titleLettersTotal = this.DOM.titleLetters.length;
    }
}

class Slideshow {
    constructor(el) {
        this.DOM = {el: el};
        this.slides = [];
        Array.from(this.DOM.el.querySelectorAll('.slide')).forEach(slide => this.slides.push(new Slide(slide)));
        this.slidesTotal = this.slides.length;
        this.current = 0;
        this.slides[this.current].DOM.el.classList.add('slide--current');

        this.DOM.el.querySelector('.reset__button').disabled = true;

        this.navigationCtrls = {
            translate: this.DOM.el.querySelector('.translate__button'),
            reset: this.DOM.el.querySelector('.reset__button')
        };
        this.initEvents();
    }

    refresh(translation) {
        var title = this.slides[0].DOM.el.querySelector('.slide__title');
        var text = reconstructText(title);
        deleteSpans(title);
        title.textContent = text;

        var title = this.slides[1].DOM.el.querySelector('.slide__title');
        deleteSpans(title);
        title.textContent = translation;

        this.slides = [];
        Array.from(this.DOM.el.querySelectorAll('.slide')).forEach(slide => this.slides.push(new Slide(slide)));
        this.slidesTotal = this.slides.length;
    }
    initEvents() {
        this.navigationCtrls.translate.addEventListener('click', () => this.navigate('next'));
        this.navigationCtrls.reset.addEventListener('click', () => this.navigate('prev'));
    }
    navigate(direction = 'next') {
        if ( this.isAnimating ) return;
        this.isAnimating = true;

        if (direction == 'next'){
            var title = this.slides[0].DOM.el.querySelector('.slide__title');
            var text = reconstructText(title);
            var url = "/translateQuery/" + encodeURIComponent(text);
            var raw = httpGet(url);
            var reply = JSON.parse(raw);
            this.refresh(reply["Translation"]);
        }

        const currentSlide = this.slides[this.current];
        this.current = direction === 'next' ? this.current < this.slidesTotal - 1 ? this.current+1 : 0 :
                    this.current > 0 ? this.current-1 : this.slidesTotal - 1;
        const upcomingSlide = this.slides[this.current];

        // The elements we will animate.
        const currentTitle = currentSlide.DOM.title;
        const currentTitleLetters = currentSlide.DOM.titleLetters;
        const currentTitleLettersTotal = currentSlide.titleLettersTotal;
        const upcomingTitle = upcomingSlide.DOM.title;

        this.tl = new TimelineMax({
            onStart: () => {
                upcomingSlide.DOM.el.classList.add('slide--current');
                upcomingSlide.DOM.el.classList.remove('slide--hidden');
            },
            onComplete: () => {
                currentSlide.DOM.el.classList.add('slide--hidden');
                currentSlide.DOM.el.classList.remove('slide--current');
                this.isAnimating = false;

                if (direction == 'next'){
                    this.DOM.el.querySelector('.translate__button').disabled = true;
                    this.DOM.el.querySelector('.reset__button').disabled = false;
                } else {
                    this.DOM.el.querySelector('.translate__button').disabled = false;
                    this.DOM.el.querySelector('.reset__button').disabled = true;
                }
            }
        }).add('begin');

        this.tl
        .set(upcomingTitle, {x: direction === 'next' ? 600 : -600, y: 0, opacity: 0})
        .to(currentTitle, 0.1, {
            ease: Quad.easeOut,
            x: direction === 'next' ? 8 : -8,
            y: 2,
            repeat: 9
        }, 'begin+=0.2')
        .staggerTo(currentTitleLetters.sort((a,b) => 0.5 - Math.random()), 0.2, {
            ease: Expo.easeOut,
            cycle: {
                x: () => direction === 'next' ? getRandomNumber(-800, -400) : getRandomNumber(400, 800),
                y: () => getRandomNumber(-40, 40),
            },
            opacity: 0
        }, 0.5/currentTitleLettersTotal, 'begin+=0.6')
        .to(upcomingTitle, 0.6, {
            ease: Elastic.easeOut.config(1,0.7),
            x: 0,
            opacity: 1
        }, 'begin+=1.15')
        .set(currentTitleLetters, {x: 0, y: 0, opacity: 1});

        this.tl.addCallback(() => {
            document.getElementById("translation-motion").classList.add('show-deco');
        }, 'begin+=0.2');

        this.tl.addCallback(() => {
            document.getElementById("translation-motion").classList.remove('show-deco');
        }, 'begin+=1.1');
    }
}