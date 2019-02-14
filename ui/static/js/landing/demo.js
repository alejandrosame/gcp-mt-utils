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

// Generates a random float.
const getRandom = (min, max) => (Math.random() * (max - min) + min).toFixed(2);

// from http://www.quirksmode.org/js/events_properties.html#position
const getMousePos = (e) => {
    let posx = 0;
    let posy = 0;
    if (!e) e = window.event;
    if (e.pageX || e.pageY) 	{
        posx = e.pageX;
        posy = e.pageY;
    }
    else if (e.clientX || e.clientY) 	{
        posx = e.clientX + document.body.scrollLeft + document.documentElement.scrollLeft;
        posy = e.clientY + document.body.scrollTop + document.documentElement.scrollTop;
    }
    return { x : posx, y : posy }
};

// Equation of a line (y = mx + b ).
const lineEq = (y2, y1, x2, x1, currentVal) => {
    const m = (y2 - y1) / (x2 - x1);
    const b = y1 - m * x1;
    return m * currentVal + b;
};

// Window sizes.
let winsize;
const calcWinsize = () => winsize = {width: window.innerWidth, height: window.innerHeight};
calcWinsize();
window.addEventListener('resize', calcWinsize);

let allowTilt = true;

class Grid {
    constructor(el) {
        this.DOM = {el: el};
        // Some configuration values.
        this.config = {
            // The min and max values to move each item (y-axis) when we move the mouse.
            titltOffset: {min: 5, max: 40}
        };
        // Grid items.
        this.DOM.items = Array.from(this.DOM.el.querySelectorAll('.grid__item'));
        // Total items.
        this.itemsTotal = this.DOM.items.length;
        // Spread the grid items.
        this.spread();
		window.addEventListener('resize', () => this.spread());
    }
    spread() {
        // Randomly spread the grid items.
        this.DOM.items.forEach((item) => {
            // The min and max values to move each item (y-axis) when we move the mouse.
            const randnum = getRandom(this.config.titltOffset.min,this.config.titltOffset.max);
            item.dataset.minTy = -1*randnum;
            item.dataset.maxTy = randnum;

            const rect = item.getBoundingClientRect();
            // Item´s center point.
            const center = {x: rect.left+rect.width/2, y: rect.top+rect.height/2};
            // Calculate the item´s quadrant in the viewport.
            const quadrant = center.x >= winsize.width/2 ?
                                center.y <= winsize.height/2 ? 1 : 4 :
                                center.y <= winsize.height/2 ? 2 : 3;
            
            // Now calculate how much to translate the item.
            // The positions will be random but only in the area of the item´s quadrant.
            // Also, consider a margin so the item does not stay completely out of the viewport or its quadrant.
            const margins = {x: winsize.width*.05, y: winsize.height*.05}
            const tx = quadrant === 1 || quadrant === 4 ? 
                    getRandom(-1*center.x + winsize.width/2 + margins.x, winsize.width - center.x - margins.x) :
                    getRandom(-1*center.x + margins.x, winsize.width/2 - center.x - margins.x);
            const ty = quadrant === 1 || quadrant === 2 ?
                    getRandom(-1*center.y + margins.y, winsize.height/2 - center.y - margins.y) :
                    getRandom(-1*center.y + winsize.height/2 + margins.y, winsize.height - center.y - margins.y);

            // Save the current translation.
            item.dataset.ctx = tx;
            item.dataset.cty = ty;

            TweenMax.set(item, {
                x: tx,
                y: ty,
                scale: 0.5
            });
        });
    }
    tilt(ev) {
        if ( !allowTilt ) return;
        const mousepos = getMousePos(ev);
        // Document scrolls.
        const docScrolls = {
            left : document.body.scrollLeft + document.documentElement.scrollLeft, 
            top : document.body.scrollTop + document.documentElement.scrollTop
        };
        // Mouse position relative to the main element.
        const relmousepos = { 
            x : mousepos.x - docScrolls.left, 
            y : mousepos.y - docScrolls.top 
        };
        // Movement settings for the tilt elements.
        this.DOM.items.forEach((item) => {
            TweenMax.to(item, 4, {
                ease: Quint.easeOut,
                y: Number(item.dataset.cty) + lineEq(item.dataset.maxTy,item.dataset.minTy,winsize.height,0,relmousepos.y)
            });
        });
    }
    hideItems(direction) {
        return this.toggleItems('hide', direction);
    }
    showItems(direction) {
        return this.toggleItems('show', direction);
    }
    toggleItems(action, direction) {
        return new Promise((resolve, reject) => {
            let cnt = 0;
            this.DOM.items.forEach((item) => {
                const rect = item.getBoundingClientRect();
                
                // The speed and delay will depend on how much the item can be translated when moving the mouse (this.config.titltOffset).
                // This will result in some items moving faster than others and also starting at different times.
                const speed = lineEq(1.3,0.9,this.config.titltOffset.min,this.config.titltOffset.max,item.dataset.maxTy);
                const delay = lineEq(0,0.4,this.config.titltOffset.min,this.config.titltOffset.max,item.dataset.maxTy);

                TweenMax.to(item, speed, {
                    ease: Expo.easeInOut,
                    delay: delay,
                    startAt: action === 'show' ? {y: direction === 'up' ? `+=${winsize.height + rect.height}` : `-=${winsize.height + rect.height}`, opacity:1} : null,
                    y: action === 'show' ? item.dataset.cty : 
                        direction === 'up' ? `-=${winsize.height + rect.height}` : `+=${winsize.height + rect.height}`
                });

                TweenMax.to(item, action === 'show' ? speed*.55 : speed*.45, {
                    ease: action === 'show' ? Quad.easeIn : Expo.easeIn,
                    delay: delay,
                    scaleX: 0.45,
                    scaleY: getRandom(1,1.3),
                    opacity: 0.5,
                    onComplete: () => {
                        TweenMax.to(item, action === 'show' ? speed*.45 : speed*.55, {
                            ease: action === 'show' ? Expo.easeOut : Quad.easeOut,
                            scaleX: 0.5,
                            scaleY: 0.5,
                            opacity: 1,
                            onComplete: () => {
                                if ( action === 'hide' ) {
                                    TweenMax.set(item, {opacity: 0, y: item.dataset.cty});
                                }
                                cnt++;
                                if ( this.itemsTotal === cnt ) {
                                    resolve();
                                }

                                // If we want to shuffle the items again after the navigation:
                                if ( action === 'hide' ) {
                                    TweenMax.set(item, {opacity: 0, y: 0});
                                }
                                cnt++;
                                if ( this.itemsTotal === cnt ) {
                                    if ( action === 'hide' ) {
                                        this.spread();
                                    }
                                    resolve();
                                }
                            }
                        });    
                    }
                });
            });
        });
    }
}

class MenuItem {
    constructor(el) {
        this.DOM = {el: el};
        this.DOM.number = this.DOM.el.querySelector('.menu__item-number');
        this.DOM.textwrap = this.DOM.el.querySelector('.menu__item-textwrap');
        this.DOM.text = this.DOM.textwrap.querySelector('.menu__item-text');
        this.DOM.link = this.DOM.el.querySelector('.menu__item-link');
    }
    toggleCurrent(direction = 'up') {
        const isCurrent = this.DOM.el.classList.contains('menu__item--current');
        this.DOM.el.classList[isCurrent ? 'remove' : 'add']('menu__item--current');
        // Toggle the link element ("explore").
        
        TweenMax.to(this.DOM.link, 1, {
            ease: Expo.easeOut,
            startAt: isCurrent ? null : {opacity: 0, y: direction === 'up' ? 15 : -15},
            y: isCurrent ? direction === 'up' ? -15 : 15 : 0,
            opacity: isCurrent ? 0 : 1
        });
        
    }
    show() {
        this.toggle('show');
    }
    hide() {
        this.toggle('hide');
    }
    toggle(action) {
        // Slide in/out the text.
        TweenMax.to(this.DOM.text, action === 'hide' ? 0.5 : 1, {
            ease: action === 'hide' ? Expo.easeIn : Expo.easeInOut,
            startAt: action === 'hide' ? null : {y: '103%'},
            y: action === 'hide' ? '103%' : '0%'
        });
        
        // Fade in/out the number and link.
        let extraElems = [this.DOM.number, this.DOM.link];
        if ( action === 'show' && !this.DOM.el.classList.contains('menu__item--current') ) {
            extraElems = [this.DOM.number];
        }
        TweenMax.to(extraElems, action === 'hide' ? 0.5 : 1, {
            ease: action === 'hide' ? Quint.easeIn : Quint.easeInOut,
            startAt: action === 'hide' ? null : {opacity: 0},
            opacity: action === 'hide' ? 0 : 1
        });
    }
}

class NavController {
    constructor(el) {
        this.DOM = {menu: el}; // Initialize with the menu element.
        // The Menu items instances.
        this.menuItems = [];
        Array.from(this.DOM.menu.querySelectorAll('.menu__item')).forEach((item) => this.menuItems.push(new MenuItem(item)));
        // The grid´s wrap.
        this.DOM.gridWrap = document.querySelector('.gridwrap');
        // The grid instances.
        this.grids = [];
        Array.from(this.DOM.gridWrap.querySelectorAll('.grid')).forEach((grid) => this.grids.push(new Grid(grid)));
        this.init();
    }
    init() {
        // Current nav menu item index (starting with the first one).
        this.current = 0;
        // Add current class to the first menu item.
        this.menuItems[this.current].toggleCurrent();
        // Also show the current grid items.
        this.grids[this.current].DOM.items.forEach((item) => TweenMax.set(item, {opacity: 1}));

        this.initEvents();
    }
    initEvents() {
        // Move the current grid´s items on the y-axis as the user moves the mouse.
        this.mousemoveFn = (ev) => requestAnimationFrame(() => this.grids[this.current].tilt(ev));
        window.addEventListener('mousemove', this.mousemoveFn);

        // Clicking the menu item text and link. (navigation and show the grid/content).
        for (const [pos, item] of this.menuItems.entries()) {
            // Clicking on the menu item text will trigger the navigation: the current grid items move away and the new ones come in.
            item.DOM.textwrap.addEventListener('click', () => this.navigate(pos));
        }
    }
    navigate(pos) {
        if ( this.isAnimating || pos === this.current ) return;
        const direction = this.current < pos ? 'up' : 'down';
		this.menuItems[this.current].toggleCurrent(direction);
        this.isAnimating = true;
        // Disable the mousemove functionality.
        allowTilt = false;
        // Hide the current grid items.
        this.grids[this.current].hideItems(direction);
        // Update current value.
        this.current = pos;
		this.menuItems[this.current].toggleCurrent(direction);
		// Show the next grid items.
        this.grids[this.current].showItems(direction).then(() => {
            this.isAnimating = false;
            allowTilt = true;
        });
    }
}
