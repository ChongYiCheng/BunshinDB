if (document.readyState == 'loading') {
        document.addEventListener('DOMContentLoaded', ready)
} else {
        ready()
}

function ready() {
        var viewCartButtons = document.getElementsByClassName('view-cart')
        for (var i = 0; i < viewCartButtons.length; i++) {
                    var button = viewCartButtons[i]
                    button.addEventListener('click', showCart)
                }
        
        var removeCartItemButtons = document.getElementsByClassName('btn-danger')
        for (var i = 0; i < removeCartItemButtons.length; i++) {
                    var button = removeCartItemButtons[i]
                    button.addEventListener('click', removeCartItem)
                }

        var quantityInputs = document.getElementsByClassName('cart-quantity-input')
        for (var i = 0; i < quantityInputs.length; i++) {
                    var input = quantityInputs[i]
                    input.addEventListener('change', quantityChanged)
                }

        var addToCartButtons = document.getElementsByClassName('shop-item-button buy-button')
        for (var i = 0; i < addToCartButtons.length; i++) {
                    var button = addToCartButtons[i]
                    button.addEventListener('click', addToCartClicked)
                }

        document.getElementsByClassName('btn-purchase')[0].addEventListener('click', purchaseClicked)

//ModalStuff 
    console.log("Modal")
	function ModalSignin( element ) {
		this.element = element;
		this.blocks = this.element.getElementsByClassName('js-signin-modal-block');
		this.switchers = this.element.getElementsByClassName('js-signin-modal-switcher')[0].getElementsByTagName('a'); 
		this.triggers = document.getElementsByClassName('js-signin-modal-trigger');
		this.hidePassword = this.element.getElementsByClassName('js-hide-password');
		this.init();
	};

	ModalSignin.prototype.init = function() {
		var self = this;
        console.log("Init Modal")
		//open modal/switch form
		for(var i =0; i < this.triggers.length; i++) {
			(function(i){
				self.triggers[i].addEventListener('click', function(event){
					if( event.target.hasAttribute('data-signin') ) {
						event.preventDefault();
						self.showSigninForm(event.target.getAttribute('data-signin'));
					}
				});
			})(i);
		}

		//close modal
		this.element.addEventListener('click', function(event){
			if( hasClass(event.target, 'js-signin-modal') || hasClass(event.target, 'js-close') ) {
				event.preventDefault();
				removeClass(self.element, 'cd-signin-modal--is-visible');
			}
		});
		//close modal when clicking the esc keyboard button
		document.addEventListener('keydown', function(event){
			(event.which=='27') && removeClass(self.element, 'cd-signin-modal--is-visible');
		});

		//hide/show password
		for(var i =0; i < this.hidePassword.length; i++) {
			(function(i){
				self.hidePassword[i].addEventListener('click', function(event){
					self.togglePassword(self.hidePassword[i]);
				});
			})(i);
		} 

		//IMPORTANT - REMOVE THIS - it's just to show/hide error messages in the demo
		this.blocks[0].getElementsByTagName('form')[0].addEventListener('submit', function(event){
			event.preventDefault();
            var stuff = document.getElementById('signin-email').value
            var user = document.getElementById('Shop_User')
            console.log(user)
            user.innerHTML = stuff;
            console.log(stuff)
			//self.toggleError(document.getElementById('signin-email'), true);
		});
		this.blocks[1].getElementsByTagName('form')[0].addEventListener('submit', function(event){
			event.preventDefault();
			self.toggleError(document.getElementById('signup-username'), true);
		});
	};

	ModalSignin.prototype.togglePassword = function(target) {
		var password = target.previousElementSibling;
		( 'password' == password.getAttribute('type') ) ? password.setAttribute('type', 'text') : password.setAttribute('type', 'password');
		target.textContent = ( 'Hide' == target.textContent ) ? 'Show' : 'Hide';
		putCursorAtEnd(password);
	}

	ModalSignin.prototype.showSigninForm = function(type) {
		// show modal if not visible
		!hasClass(this.element, 'cd-signin-modal--is-visible') && addClass(this.element, 'cd-signin-modal--is-visible');
		// show selected form
		for( var i=0; i < this.blocks.length; i++ ) {
			this.blocks[i].getAttribute('data-type') == type ? addClass(this.blocks[i], 'cd-signin-modal__block--is-selected') : removeClass(this.blocks[i], 'cd-signin-modal__block--is-selected');
		}
		//update switcher appearance
		var switcherType = (type == 'signup') ? 'signup' : 'login';
		for( var i=0; i < this.switchers.length; i++ ) {
			this.switchers[i].getAttribute('data-type') == switcherType ? addClass(this.switchers[i], 'cd-selected') : removeClass(this.switchers[i], 'cd-selected');
		} 
	};

	ModalSignin.prototype.toggleError = function(input, bool) {
		// used to show error messages in the form
		toggleClass(input, 'cd-signin-modal__input--has-error', bool);
		toggleClass(input.nextElementSibling, 'cd-signin-modal__error--is-visible', bool);
	}

	var signinModal = document.getElementsByClassName("js-signin-modal")[0];
	if( signinModal ) {
		new ModalSignin(signinModal);
	}

	// toggle main navigation on mobile
	var mainNav = document.getElementsByClassName('js-main-nav')[0];
	if(mainNav) {
		mainNav.addEventListener('click', function(event){
			if( hasClass(event.target, 'js-main-nav') ){
				var navList = mainNav.getElementsByTagName('ul')[0];
				toggleClass(navList, 'cd-main-nav__list--is-visible', !hasClass(navList, 'cd-main-nav__list--is-visible'));
			} 
		});
	}
	
	//class manipulations - needed if classList is not supported
	function hasClass(el, className) {
	  	if (el.classList) return el.classList.contains(className);
	  	else return !!el.className.match(new RegExp('(\\s|^)' + className + '(\\s|$)'));
	}
	function addClass(el, className) {
		var classList = className.split(' ');
	 	if (el.classList) el.classList.add(classList[0]);
	 	else if (!hasClass(el, classList[0])) el.className += " " + classList[0];
	 	if (classList.length > 1) addClass(el, classList.slice(1).join(' '));
	}
	function removeClass(el, className) {
		var classList = className.split(' ');
	  	if (el.classList) el.classList.remove(classList[0]);	
	  	else if(hasClass(el, classList[0])) {
	  		var reg = new RegExp('(\\s|^)' + classList[0] + '(\\s|$)');
	  		el.className=el.className.replace(reg, ' ');
	  	}
	  	if (classList.length > 1) removeClass(el, classList.slice(1).join(' '));
	}
	function toggleClass(el, className, bool) {
		if(bool) addClass(el, className);
		else removeClass(el, className);
	}

	//credits http://css-tricks.com/snippets/jquery/move-cursor-to-end-of-textarea-or-input/
	function putCursorAtEnd(el) {
    	if (el.setSelectionRange) {
      		var len = el.value.length * 2;
      		el.focus();
      		el.setSelectionRange(len, len);
    	} else {
      		el.value = el.value;
    	}
	};

}

function showCart(event) {
    var container = document.getElementsByClassName('container')[0]
    //var x = document.getElementById("myDIV");
    if (container.style.display === "none") {
        container.style.display = "block";
    } else {
        container.style.display = "none";
    }
}

function addToCartClicked(event) {
    var button = event.target
    var shopItem = button.parentElement.parentElement
    var title = shopItem.getElementsByClassName('product__name')[0].innerText
    var price = shopItem.getElementsByClassName('product__price')[0].innerText
    var imageSrc = shopItem.getElementsByClassName('product__image')[0].src
	console.log(title)
	console.log(price)
    addItemToCart(title, price, imageSrc)
    updateCartTotal()
}

function addItemToCart(title, price, imageSrc) {
    var cartRow = document.createElement('div')
    cartRow.classList.add('cart-row')
    var cartItems = document.getElementsByClassName('cart-items')[0]
    var cartItemNames = cartItems.getElementsByClassName('cart-item-title')
    for (var i = 0; i < cartItemNames.length; i++) {
        if (cartItemNames[i].innerText == title) {
            alert('This item is already added to the cart')
            return
        }
    }
    var cartRowContents = `
        <div class="cart-item cart-column">
            <img class="cart-item-image" src="${imageSrc}" width="100" height="100">
            <span class="cart-item-title">${title}</span>
        </div>
        <span class="cart-price cart-column">${price}</span>
        <div class="cart-quantity cart-column">
            <input class="cart-quantity-input" type="number" value="1">
            <button class="btn btn-danger" type="button">REMOVE</button>
        </div>`
    cartRow.innerHTML = cartRowContents
    cartItems.append(cartRow)
    cartRow.getElementsByClassName('btn-danger')[0].addEventListener('click', removeCartItem)
    cartRow.getElementsByClassName('cart-quantity-input')[0].addEventListener('change', quantityChanged)
}

function updateCartTotal() {
    console.log("Update Cart activated")
    var cartItemContainer = document.getElementsByClassName('cart-items')[0]
    var cartRows = cartItemContainer.getElementsByClassName('cart-row')
    var total = 0
    for (var i = 0; i < cartRows.length; i++) {
        var cartRow = cartRows[i]
        var priceElement = cartRow.getElementsByClassName('cart-price')[0]
        var quantityElement = cartRow.getElementsByClassName('cart-quantity-input')[0]
        var price = parseFloat(priceElement.innerText.replace('$', ''))
        var quantity = quantityElement.value
        console.log("Checking updateCartTotal")
        console.log(price)
        console.log(quantity)
        total = total + (price * quantity)
    }
    total = Math.round(total * 100) / 100
    var cartTotalPrice = document.getElementsByClassName('cart-total-price')
    for (var i = 0; i < cartTotalPrice.length; i++) {
        cartTotalPrice[i].innerText = '$' + total
    }
}

function removeCartItem(event) {
    var buttonClicked = event.target
    buttonClicked.parentElement.parentElement.remove()
    updateCartTotal()
}

function quantityChanged(event) {
    var input = event.target
    if (isNaN(input.value) || input.value <= 0) {
        input.value = 1
    }
    updateCartTotal()
}

function purchaseClicked() {
    alert('Thank you for your purchase')
    var cartItems = document.getElementsByClassName('cart-items')[0]
    while (cartItems.hasChildNodes()) {
        cartItems.removeChild(cartItems.firstChild)
    }
    updateCartTotal()
}

//TODO Make a function with AJAX to send JSON to client.go



//TODO Make a function with AJAX to receive JSON from client.go

//Modal JS
//
//

//(function(){
//    //Login/Signup modal window - by CodyHouse.co
//})();
