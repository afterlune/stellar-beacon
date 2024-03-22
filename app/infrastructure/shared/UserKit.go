package shared

import "benetnasch/app/facade/model"

var userDetailsDTO model.UserDetailsDTO

func SetUserDetails(dto model.UserDetailsDTO) {
	userDetailsDTO = dto
}

func GetUserDetailsDTO() model.UserDetailsDTO {
	return userDetailsDTO
}
